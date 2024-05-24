package api

import (
	"context"
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
		w.WriteHeader(422)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			return
		}
		return
	}
	cep := data.Cep

	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := tracer.Start(ctx, "request-service-a")
	defer span.End()

	url := "http://service-b:8081/" + cep
	response, err := fetchData(ctx, url)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		w.WriteHeader(500)
		_, err := w.Write([]byte("internal server error"))
		if err != nil {
			return
		}
		return
	}

	w.WriteHeader(200)
	_, err = w.Write(response)
	if err != nil {
		return
	}
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
