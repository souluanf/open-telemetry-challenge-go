package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type CepRequest struct {
	Cep string `json:"cep"`
}

func (req *CepRequest) Bind(r *http.Request) error {
	if req.Cep == "" {
		return errors.New("cep field is missing")
	}

	req.Cep = strings.Replace(req.Cep, "-", "", -1)

	if len(req.Cep) != 8 {
		return errors.New("invalid zipcode")
	}
	return nil
}

var tracer = otel.Tracer("open-telemetry-challenge-go")

func HandleRequest(w http.ResponseWriter, r *http.Request) {
	data := &CepRequest{}
	if err := render.Bind(r, data); err != nil {
		writeErrorResponse(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	cep := data.Cep

	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := tracer.Start(ctx, "request-service-a")
	defer span.End()

	url := "http://service-b:8081/" + cep
	statusCode, response, err := fetchData(ctx, url)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(statusCode)
	_, err = w.Write(response)
	if err != nil {
		return
	}
}

func fetchData(c context.Context, url string) (int, []byte, error) {
	res, err := otelhttp.Get(c, url)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	return res.StatusCode, body, nil
}

func writeErrorResponse(w http.ResponseWriter, code int, message string) {
	errorResponse := map[string]interface{}{
		"statuscode": code,
		"message":    message,
	}
	response, _ := json.Marshal(errorResponse)
	w.WriteHeader(code)
	_, _ = w.Write(response)
}
