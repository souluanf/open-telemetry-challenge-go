package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	urlViaCep     = "https://viacep.com.br/"
	urlWeatherApi = "https://api.weatherapi.com/v1/"
	weatherApiKey = "465c66df5be547d790a181453242405"
)

var tracer = otel.Tracer("open-telemetry-challenge-go")

func HandleRequest(w http.ResponseWriter, r *http.Request) {
	cep := chi.URLParam(r, "cep")
	cep = strings.Replace(cep, "-", "", -1)

	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := tracer.Start(ctx, "request-service-b")
	defer span.End()

	if !validateCep(cep, w) {
		return
	}

	cepResponse, err := getCepResponse(ctx, cep)
	if err != nil {
		handleError(w, "Error fetching zipcode data", 500, err)
		return
	}

	if cepResponse.Erro == "true" {
		handleError(w, "can not find zipcode", 404, nil)
		return
	}

	city, state := removeAccents(cepResponse.Localidade), cepResponse.Uf
	weatherResponse, err := getWeatherResponse(ctx, city, state)
	if err != nil {
		handleError(w, "internal server error", 500, err)
		return
	}

	writeResponse(w, weatherResponse, city, state)
}

func validateCep(cep string, w http.ResponseWriter) bool {
	if len(cep) != 8 {
		w.WriteHeader(422)
		_, _ = w.Write([]byte("invalid zipcode"))
		return false
	}
	return true
}

func getCepResponse(ctx context.Context, cep string) (ViaCepResponse, error) {
	url := urlViaCep + "ws/" + cep + "/json"
	respCep, err := fetchData(ctx, url)
	if err != nil {
		return ViaCepResponse{}, err
	}

	var cepResponse ViaCepResponse
	err = json.Unmarshal(respCep, &cepResponse)
	return cepResponse, err
}

func getWeatherResponse(ctx context.Context, city, state string) (WeatherApiResponse, error) {
	url := urlWeatherApi + "current.json?key=" + weatherApiKey + "&q=" + city + " - " + state + " - Brazil&aqi=no&tides=no"
	url = strings.Replace(url, " ", "%20", -1)

	respWeather, err := fetchData(ctx, url)
	if err != nil {
		return WeatherApiResponse{}, err
	}

	var weatherResponse WeatherApiResponse
	err = json.Unmarshal(respWeather, &weatherResponse)
	return weatherResponse, err
}

func writeResponse(w http.ResponseWriter, weatherResponse WeatherApiResponse, city, state string) {
	tempC := strconv.FormatFloat(weatherResponse.Current.TempC, 'f', -1, 64)
	tempF := strconv.FormatFloat(weatherResponse.Current.TempF, 'f', -1, 64)
	tempK := strconv.FormatFloat(weatherResponse.Current.TempC+273.15, 'f', -1, 64)

	response := []byte(fmt.Sprintf(`{ "city": "%s/%s", "temp_C": %s, "temp_F": %s, "temp_K": %s }`, city, state, tempC, tempF, tempK))

	w.WriteHeader(200)
	_, _ = w.Write(response)
}

func handleError(w http.ResponseWriter, message string, code int, err error) {
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}
	w.WriteHeader(code)
	_, _ = w.Write([]byte(message))
}

func fetchData(c context.Context, url string) (response []byte, err error) {
	res, _ := otelhttp.Get(c, url)
	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return body, nil
}

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, e := transform.String(t, s)
	if e != nil {
		panic(e)
	}
	return output
}
