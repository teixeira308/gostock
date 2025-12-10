📚 GoStock API
Visão Geral do Projeto
GoStock é um projeto de API construído em Go (Golang), seguindo os princípios da Arquitetura Limpa (Clean Architecture). O objetivo é fornecer uma solução robusta e escalável para gerenciamento de catálogo de produtos, estoque e transações, utilizando PostgreSQL como banco de dados principal e Redis para caching de alto desempenho.

🏗️ Arquitetura
O projeto é estruturado em camadas para garantir separação de responsabilidades, testabilidade e manutenibilidade (Clean Architecture).

| Camada       | Responsabilidade                                                                      | Pacotes                        |
|--------------|---------------------------------------------------------------------------------------|--------------------------------|
| Domain       | O Core do negócio: entidades (Product, Variant, Warehouse, StockLevel), interfaces de serviço e repositório. | `internal/domain`              |
| Service      | Regras de Negócio e Orquestração (ex: validação de SKU, criação de ID, ajuste de estoque). | `internal/service/*`           |
| API          | Entrada HTTP e Despacho: decodifica requisições, chama o Service, formata respostas.     | `internal/api/*`               |
| Repository   | Acesso a dados: implementa interfaces do Domain, manipula DB (PostgreSQL) e Cache (Redis). | `internal/repository/*`        |
| Infrastructure | Inicialização, Configuração e Conexões (DB, Cache, Router).                         | `internal/infrastructure/*`, `cmd/main.go` |

⚙️ Configuração de Ambiente
Este projeto requer Docker Desktop para rodar os serviços de infraestrutura (PostgreSQL e Redis).

### Variáveis de Ambiente
O projeto utiliza o arquivo `.env` (localizado na raiz, não versionado) para carregar as credenciais. Certifique-se de que este arquivo existe e contém as seguintes variáveis:

```
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

# Configuração de Segurança JWT
JWT_SECRET_KEY=sua_chave_secreta_aqui # MUDE ISTO EM PRODUÇÃO!
JWT_EXPIRY_HOURS=24

# Nível de Log (debug, info, warn, error, fatal)
LOG_LEVEL=info

# Variável de URL para Migrações (Goose)
# IMPORTANTE: Formato 'postgres://user:password@host:port/dbname?sslmode=disable'
DATABASE_URL=postgres://user:password@localhost:5432/gostock_db?sslmode=disable
```

### Serviços Docker
Execute os seguintes comandos no terminal para subir o PostgreSQL e o Redis:

1.  **PostgreSQL (DB Principal)**
    ```bash
    docker run --name gostock-postgres \
    -e POSTGRES_DB=gostock_db \
    -e POSTGRES_USER=user \
    -e POSTGRES_PASSWORD=password \
    -p 5432:5432 \
    -d postgres:15-alpine
    ```
2.  **Redis (Cache)**
    ```bash
    docker run --name gostock-redis \
    -p 6379:6379 \
    -d redis:7-alpine
    ```

🗄️ Migrações de Banco de Dados (Goose)
Utilizamos o Goose para gerenciar o schema do banco de dados.

### Passos para Rodar Migrações:

1.  **Compilar e Executar o Migrador:**
    Para aplicar as migrações pendentes (como a criação das tabelas `warehouses` e `stock_levels` e a extensão `uuid-ossp`), utilize o executável do Goose que foi configurado no projeto:
    ```bash
    go run cmd/migrate/main.go up
    ```
    Este comando lerá o `DATABASE_URL` do seu ambiente, conectará ao PostgreSQL e aplicará todas as migrações `.sql` necessárias encontradas na pasta `./sql`.

---

▶️ Executando o Projeto
Com os serviços Docker e as migrações aplicadas, execute o servidor Go:

```bash
go run cmd/main.go
```
O servidor estará disponível em `http://localhost:8080`.

---

🧪 Funcionalidades Implementadas (Testadas via Postman/Curl)
As seguintes endpoints e funcionalidades foram implementadas, cobrindo o fluxo completo de gerenciamento de produtos, estoque e armazéns.

### 1. 👤 Autenticação e Autorização (JWT)
A API implementa um sistema de segurança baseado em JSON Web Tokens (JWT) para proteger endpoints sensíveis.

**Fluxo de Autenticação:**
1.  **Registro:** Um novo usuário é criado através do endpoint `POST /v1/register`.
2.  **Login:** O usuário se autentica com email e senha no endpoint `POST /v1/login`.
3.  **Token:** A API retorna um token JWT, que deve ser incluído no cabeçalho `Authorization` de todas as requisições subsequentes a endpoints protegidos.

**Endpoints de Autenticação:**

**a) Registrar Novo Usuário**
Cria um novo usuário no sistema.
*   **Endpoint:** `POST /v1/register`
*   **Status de Sucesso:** `201 Created`
*   **Exemplo:**
    ```bash
    curl --location 'http://localhost:8080/v1/register' \
    --header 'Content-Type: application/json' \
    --data '{
        "name": "Admin User",
        "email": "admin@gostock.com",
        "password": "strongpassword123",
        "role": "admin"
    }'
    ```

**b) Realizar Login**
Autentica o usuário e retorna um token JWT.
*   **Endpoint:** `POST /v1/login`
*   **Status de Sucesso:** `200 OK`
*   **Exemplo:**
    ```bash
    curl --location 'http://localhost:8080/v1/login' \
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

---

### 2. 📦 Produtos
Endpoints para gerenciamento do catálogo de produtos.

**a) Criar Produto (Requer Autenticação - Admin)**
Cria um produto principal e suas variantes. Este endpoint é protegido e requer um token JWT válido de um usuário `admin`.
*   **Endpoint:** `POST /v1/products`
*   **Status de Sucesso:** `201 Created`
*   **Exemplo:** (Corpo da requisição e cabeçalho Authorization conforme o Postman Collection)

**b) Obter Produto por ID (Público)**
Busca um produto específico pelo seu ID, implementando a estratégia Cache-Aside.
*   **Endpoint:** `GET /v1/products/{id}`
*   **Status de Sucesso:** `200 OK` (encontrado) ou `404 Not Found` (não encontrado).
*   **Exemplo:** (URL conforme o Postman Collection)

**c) Listar Produtos (Público)**
Lista todos os produtos com suporte a paginação e filtros.
*   **Endpoint:** `GET /v1/products`
*   **Parâmetros de Query:**
    *   `page` (opcional, int): Número da página (padrão: 1).
    *   `limit` (opcional, int): Quantidade de itens por página (padrão: 10, máximo: 100).
    *   `name` (opcional, string): Filtra produtos por nome (case-insensitive, busca parcial).
    *   `sku` (opcional, string): Filtra produtos por SKU (busca exata).
    *   `active_only` (opcional, boolean): `true` para listar apenas produtos ativos.
*   **Status de Sucesso:** `200 OK`
*   **Exemplo:** (URL conforme o Postman Collection)

---

### 3. 🏢 Armazéns
Endpoints para gerenciamento de armazéns.

**a) Criar Armazém (Requer Autenticação - Admin)**
Cria um novo armazém.
*   **Endpoint:** `POST /v1/warehouses`
*   **Status de Sucesso:** `201 Created`
*   **Exemplo:** (Corpo da requisição conforme `api_body_examples.md`)

**b) Obter Armazém por ID (Público)**
Busca um armazém específico pelo seu ID.
*   **Endpoint:** `GET /v1/warehouses/{id}`
*   **Status de Sucesso:** `200 OK` ou `404 Not Found`.

**c) Listar Todos os Armazéns (Público)**
Lista todos os armazéns cadastrados.
*   **Endpoint:** `GET /v1/warehouses`
*   **Status de Sucesso:** `200 OK`

**d) Atualizar Armazém (Requer Autenticação - Admin)**
Atualiza os dados de um armazém existente.
*   **Endpoint:** `PUT /v1/warehouses/{id}`
*   **Status de Sucesso:** `200 OK`
*   **Exemplo:** (Corpo da requisição conforme `api_body_examples.md`)

**e) Deletar Armazém (Requer Autenticação - Admin)**
Remove um armazém pelo seu ID.
*   **Endpoint:** `DELETE /v1/warehouses/{id}`
*   **Status de Sucesso:** `204 No Content`

---

### 4. 📈 Estoque
Endpoints para gerenciamento do nível de estoque.

**a) Ajustar Nível de Estoque (Requer Autenticação - Admin)**
Ajusta a quantidade de estoque para uma `variantID` em um `warehouseID` específico. Implementa **Transações SQL** e **Controle de Concorrência Otimista (OCC)**.
*   **Endpoint:** `POST /v1/stock/update`
*   **Status de Sucesso:** `200 OK` (ajuste) ou `201 Created` (inserção inicial).
*   **Status de Erro Notáveis:** `400 Bad Request` (estoque negativo, payload inválido), `409 Conflict` (OCC falhou).
*   **Exemplo:** (Corpo da requisição conforme `api_body_examples.md`)

---

### 5. 🛡️ API Features

#### 5.1 Rate Limiting
A API implementa um middleware de Rate Limiting para proteger contra abusos e garantir a estabilidade do serviço.
**Como Funciona:**
*   **Baseado em IP:** O limite é aplicado por endereço IP do cliente.
*   **Armazenamento em Cache:** Utiliza o Redis para armazenar a contagem de requisições de cada IP e o tempo de expiração.
*   **Limite Atual:** Atualmente configurado para **10 requisições por minuto** por IP.
*   **Endpoints Protegidos:** As rotas de criação/gerenciamento de produtos (`/v1/products` POST), estoque (`/v1/stock/update`), armazéns (`/v1/warehouses` CRUD) e autenticação (`/v1/register`, `/v1/login`) são protegidas por Rate Limiting.
*   **Resposta:** Se o limite for excedido, a API retorna um status `429 Too Many Requests`.
*   **Headers:** As respostas incluem os seguintes cabeçalhos para informar o status do Rate Limiting: `X-RateLimit-Remaining`.

#### 5.2 Graceful Shutdown
O servidor HTTP da API está configurado para um desligamento gracioso.
**Como Funciona:**
*   **Escuta de Sinais:** O servidor ouve por sinais do sistema operacional (`SIGTERM`, `SIGINT`).
*   **Conclusão de Requisições Ativas:** Ao receber um desses sinais, o servidor tenta concluir todas as requisições ativas antes de ser completamente desligado. Isso evita interrupções abruptas para os clientes durante processos de deploy ou reinício.
*   **Implementação:** A lógica para o Graceful Shutdown reside em `cmd/main.go`, onde uma goroutine inicia o servidor e um handler de sinal captura `SIGINT` e `SIGTERM` para chamar `server.Shutdown()` com um timeout.

#### 5.3 Logging Estruturado
A API utiliza um sistema de logging estruturado e configurável para registro de eventos.
**Como Funciona:**
*   **Logger Customizado:** Implementação de um `Logger` customizado em `internal/pkg/logger/logger.go` que gera logs em formato JSON, facilitando a análise por ferramentas de observabilidade.
*   **Níveis de Log:** Suporta diversos níveis de log (`Debug`, `Info`, `Warn`, `Error`, `Fatal`) para diferentes granularidades de informação.
*   **Uso em Camadas:** O logger é injetado e utilizado extensivamente nas camadas de Handlers, Services e Repositórios para registrar o fluxo da requisição, sucesso, avisos e erros. Erros críticos (500) são registrados com detalhes para auxiliar na depuração.
*   **Configurável:** O nível de log é configurado via variável de ambiente `LOG_LEVEL` (`debug`, `info`, `warn`, `error`, `fatal`).

---

## 🛣️ Próximos Passos e Roadmap

A funcionalidade básica de Catálogo de Produtos (CRUD e Cache), gerenciamento de Estoque e Armazéns, e segurança (AuthN/AuthZ) está completa. O trabalho futuro focará em robustez e observabilidade para tornar a API pronta para produção.

### 1. 📊 Observabilidade e Monitoramento
Garantir que a aplicação seja visível e que seu desempenho possa ser rastreado.
*   **Implementação do Logger:** Concluído. A integração do **Logger** foi realizada em todas as camadas (Handlers, Services e Repositórios), garantindo o registro adequado de eventos em diferentes níveis (`Debug`, `Info`, `Warn`, `Error`, `Fatal`) para facilitar o rastreamento da causa raiz dos erros.
*   **Basic Server Metrics:** Adicionar instrumentação para coletar métricas internas (latência, contagem de erros, uso de memória) e expô-las em um *endpoint* padrão (ex: `/metrics`) para integração com **Prometheus e Grafana**.

### 2. 📝 Manutenção e Documentação
Aumentar a qualidade do código através de testes e melhorar a experiência do desenvolvedor (DX).
*   **Testing Overview:** Desenvolver testes unitários para a camada de Serviço (regras de negócio) e testes de integração para o Repositório e Handlers.
*   **Auto Generating Docs (Swagger):** Integrar ferramentas de documentação (*doc generation*) para criar uma especificação OpenAPI (Swagger) automaticamente a partir dos comentários no código, disponibilizando uma interface interativa (ex: `/swagger/index.html`).
*   **Postman Collection:** Uma coleção Postman (`gostock_postman_collection.json`) foi gerada para facilitar os testes manuais dos endpoints implementados.