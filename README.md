📚 GoStock API
Visão Geral do Projeto
GoStock é um projeto de API construído em Go (Golang), seguindo os princípios da Arquitetura Limpa (Clean Architecture). O objetivo é fornecer uma solução robusta e escalável para gerenciamento de catálogo de produtos, estoque e transações, utilizando PostgreSQL como banco de dados principal e Redis para caching de alto desempenho.

🏗️ Arquitetura
O projeto é estruturado em camadas para garantir separação de responsabilidades, testabilidade e manutenibilidade (Clean Architecture).

Camada	Responsabilidade	Pacotes
Domain	O Core do negócio: entidades (Product, Variant), interfaces de serviço e repositório.	internal/domain
Service	Regras de Negócio e Orquestração (ex: validação de SKU, criação de ID).	internal/service/*
API	Entrada HTTP e Despacho: decodifica requisições, chama o Service, formata respostas.	internal/api/*
Repository	Acesso a dados: implementa interfaces do Domain, manipula DB (PostgreSQL) e Cache (Redis).	internal/repository/*
Infrastructure	Inicialização, Configuração e Conexões (DB, Cache, Router).	internal/infrastructure/*, cmd/main.go
⚙️ Configuração de Ambiente
Este projeto requer Docker Desktop para rodar os serviços de infraestrutura (PostgreSQL e Redis).

Variáveis de Ambiente
O projeto utiliza o arquivo .env (localizado na raiz, não versionado) para carregar as credenciais. Certifique-se de que este arquivo existe e contém as seguintes variáveis:

Snippet de código
# Configuração do Banco de Dados
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=gostock_db
POSTGRES_TIMEOUT_SEC=5

# Configuração do Cache (Redis)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Variável de URL para Migrações (Goose)
# IMPORTANTE: Formato 'postgres://user:password@host:port/dbname?sslmode=disable'
DATABASE_URL=postgres://user:password@localhost:5432/gostock_db?sslmode=disable
Serviços Docker
Execute os seguintes comandos no terminal para subir o PostgreSQL e o Redis:

1. PostgreSQL (DB Principal)
Bash
docker run --name gostock-postgres \
-e POSTGRES_DB=gostock_db \
-e POSTGRES_USER=user \
-e POSTGRES_PASSWORD=password \
-p 5432:5432 \
-d postgres:15-alpine
2. Redis (Cache)
Bash
docker run --name gostock-redis \
-p 6379:6379 \
-d redis:7-alpine
🗄️ Migrações de Banco de Dados (Goose)
Utilizamos o Goose para gerenciar o schema do banco de dados.

Passo 1: Instalar o Goose
Bash
go install github.com/pressly/goose/v3/cmd/goose@latest
Verifique a instalação:

Bash
goose -version
Passo 2: Executar as Migrações Pendentes
Este comando lê o DATABASE_URL do seu ambiente e aplica todas as migrações SQL necessárias (CREATE TABLE products, CREATE TABLE variants, etc.) no PostgreSQL.

Bash
goose -dir infraestructure/migrations postgres "$DATABASE_URL" up
▶️ Executando o Projeto
Com os serviços Docker e as migrações aplicadas, execute o servidor Go:

Bash
go run cmd/main.go
O servidor estará disponível em http://localhost:8080.

🧪 Funcionalidades Implementadas (Testadas via Postman/Curl)
As seguintes endpoints foram implementadas, cobrindo o fluxo de criação e leitura do produto, desde o Handler até a persistência no DB/Cache.

1. Criar Produto (POST)
Cria um produto principal e suas variantes, garantindo a atomicidade via Transação SQL no Repositório.

Endpoint: POST /v1/products

Status de Sucesso: 201 Created

Bash
curl --location 'http://localhost:8080/v1/products' \
--header 'Content-Type: application/json' \
--data '{
    "Product": {
        "sku": "PROD-1001-XYZ",
        "name": "Smartwatch Pro X",
        "description": "Relógio inteligente com monitoramento cardíaco e GPS.",
        "price": 499.90
    },
    "Variants": [
        {
            "attribute": "Cor",
            "value": "Preto",
            "barcode": "123456789001"
        }
    ]
}'
2. Obter Produto por ID (GET)
Busca um produto, implementando a estratégia Cache-Aside (lê do Redis primeiro, salva no Redis após ler do PostgreSQL).

Endpoint: GET /v1/products/{id}

Status de Sucesso: 200 OK (encontrado) ou 404 Not Found (não encontrado).

Bash
# Substitua o ID pelo ID do produto criado
curl --location 'http://localhost:8080/v1/products/999d1263-1f11-4adb-a966-e8e4cf340a15'