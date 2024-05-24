### Execução do desafio

- Inicialize os serviços com o docker-compose
 ```bash
 docker compose up
 ```

- Faça uma requisição para o serviço
 ```bash
 curl --request POST --url 'http://localhost:8080' -H "Content-Type: application/json" -d '{"cep" : "08210010"}'
 ```

- Verifique os logs do serviço
 ```bash
 http://127.0.0.1:9411/zipkin
 ```