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
IMPORTANTE: Formato 'postgres://user:password@host:port/dbname?sslmode=disable'
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

### 1. 👤 Autenticação e Autorização (JWT)
A API implementa um sistema de segurança baseado em JSON Web Tokens (JWT) para proteger endpoints sensíveis.

**Fluxo de Autenticação:**
1.  **Registro:** Um novo usuário é criado através do endpoint `POST /v1/users/register`.
2.  **Login:** O usuário se autentica com email e senha no endpoint `POST /v1/users/login`.
3.  **Token:** A API retorna um token JWT, que deve ser incluído no cabeçalho `Authorization` de todas as requisições subsequentes a endpoints protegidos.

**Endpoints de Autenticação:**

**a) Registrar Novo Usuário**

Cria um novo usuário no sistema.

*   **Endpoint:** `POST /v1/users/register`
*   **Status de Sucesso:** `201 Created`
*   **Exemplo:**
    ```bash
    curl --location 'http://localhost:8080/v1/users/register' \
    --header 'Content-Type: application/json' \
    --data '{
        "name": "Admin User",
        "email": "admin@gostock.com",
        "password": "strongpassword123"
    }'
    ```

**b) Realizar Login**

Autentica o usuário e retorna um token JWT.

*   **Endpoint:** `POST /v1/users/login`
*   **Status de Sucesso:** `200 OK`
*   **Exemplo:**
    ```bash
    curl --location 'http://localhost:8080/v1/users/login' \
    --header 'Content-Type: application/json' \
    --data '{
        "email": "admin@gostock.com",
        "password": "strongpassword123"
    }'
    ```
    **Resposta de Sucesso (Exemplo):**
    ```json
    {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
    ```

### 2. 📦 Produtos
Endpoints para gerenciamento do catálogo de produtos.

**a) Criar Produto (Requer Autenticação)**

Cria um produto principal e suas variantes. Este endpoint é protegido e requer um token JWT válido.

*   **Endpoint:** `POST /v1/products`
*   **Status de Sucesso:** `201 Created`
*   **Exemplo:**
    ```bash
    # Substitua SEU_TOKEN_JWT pelo token obtido no login
    curl --location 'http://localhost:8080/v1/products' \
    --header 'Authorization: Bearer SEU_TOKEN_JWT' \
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
    ```

**b) Obter Produto por ID (Público)**

Busca um produto, implementando a estratégia Cache-Aside.

*   **Endpoint:** `GET /v1/products/{id}`
*   **Status de Sucesso:** `200 OK` (encontrado) ou `404 Not Found` (não encontrado).
*   **Exemplo:**
    ```bash
    # Substitua o ID pelo ID do produto criado
    curl --location 'http://localhost:8080/v1/products/999d1263-1f11-4adb-a966-e8e4cf340a15'
    ```

## 🛣️ Próximos Passos e Roadmap

A funcionalidade básica de Catálogo de Produtos (CRUD e Cache) e segurança (AuthN/AuthZ) está completa. O trabalho futuro focará em robustez e observabilidade para tornar a API pronta para produção.

### 1. 🛡️ Resiliência e Disponibilidade

Melhorar a capacidade da API de lidar com sobrecarga e garantir o desligamento seguro.

* **Rate Limiting:** Implementar um **Middleware** que utiliza o **Redis** para limitar o número de requisições por cliente (baseado em IP ou ID de usuário) dentro de um período, prevenindo abusos e ataques DoS. 
* **Graceful Shutdown:** Configurar o servidor HTTP para ouvir sinais do sistema operacional (`SIGTERM`, `SIGINT`). Isso garante que o servidor conclua as requisições ativas antes de ser desligado, evitando interrupções para o cliente durante implantações.

### 3. 📊 Observabilidade e Monitoramento

Garantir que a aplicação seja visível e que seu desempenho possa ser rastreado.

* **Implementação do Logger:** Finalizar a configuração do **Logger** em todas as camadas, garantindo o registro adequado de eventos em diferentes níveis (`Debug`, `Info`, `Error`), especialmente para rastrear a causa raiz dos erros 500.
* **Basic Server Metrics:** Adicionar instrumentação para coletar métricas internas (latência, contagem de erros, uso de memória) e expô-las em um *endpoint* padrão (ex: `/metrics`) para integração com **Prometheus e Grafana**.

### 4. 📝 Manutenção e Documentação

Aumentar a qualidade do código através de testes e melhorar a experiência do desenvolvedor (DX).

* **Testing Overview:** Desenvolver testes unitários para a camada de Serviço (regras de negócio) e testes de integração para o Repositório e Handlers.
* **Auto Generating Docs (Swagger):** Integrar ferramentas de documentação (*doc generation*) para criar uma especificação OpenAPI (Swagger) automaticamente a partir dos comentários no código, disponibilizando uma interface interativa (ex: `/swagger/index.html`).