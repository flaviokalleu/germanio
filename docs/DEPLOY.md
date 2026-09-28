# Germanio — Guia de Deploy (Deployment Guide)

> **Escopo desta referência:** Guia de operação para quem hospeda. Configuração de servidor não deve virar requisito do autor do domínio. Para a camada de intenção, consulte as variáveis de operação em [INTENCAO.md](INTENCAO.md); não presuma que nomes de variáveis de modos anteriores sejam intercambiáveis. A versão de Go exigida pelo checkout é a de `go.mod`.

> Versão 0.2.0 | Última atualização: 2026-04-09

---

## Sumário

1. [Pré-requisitos](#1-pré-requisitos)
2. [Desenvolvimento Local](#2-desenvolvimento-local)
3. [Variáveis de Ambiente](#3-variáveis-de-ambiente)
4. [SQLite — Configuração Padrão](#4-sqlite--configuração-padrão)
5. [PostgreSQL — Produção](#5-postgresql--produção)
6. [MySQL — Alternativa](#6-mysql--alternativa)
7. [Docker](#7-docker)
8. [Docker Compose](#8-docker-compose)
9. [HTTPS com Nginx (Reverse Proxy)](#9-https-com-nginx-reverse-proxy)
10. [Systemd — Serviço Linux](#10-systemd--serviço-linux)
11. [Performance e Tuning](#11-performance-e-tuning)
12. [Estratégias de Backup](#12-estratégias-de-backup)
13. [Monitoramento](#13-monitoramento)
14. [Segurança em Produção](#14-segurança-em-produção)
15. [Checklist de Deploy](#15-checklist-de-deploy)

---

## 1. Pré-requisitos

### Para desenvolvimento

| Requisito | Versão mínima | Instalação |
|---|---|---|
| Go | 1.21+ | https://go.dev/dl/ |
| Git | qualquer | https://git-scm.com |

### Para produção (compilado)

O binário `germanio` é **auto-contido** — não requer instalação de Go no servidor de produção. Apenas o binário compilado e os arquivos `.ge` são necessários.

### Compilar o binário

```bash
# Clonar o repositório
git clone https://github.com/flavio/germanio
cd germanio

# Compilar (sem CGO — binário estático)
CGO_ENABLED=0 go build -o germanio .

# Linux/Mac — também pode usar
go build -ldflags="-s -w" -o germanio .

# Windows
go build -o germanio.exe .
```

**Cross-compilation (compilar para Linux a partir de qualquer OS):**
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o germanio-linux .
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o germanio-mac .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o germanio.exe .
```

---

## 2. Desenvolvimento Local

### 2.1 Instalar o Germanio CLI

```bash
# Via go install (instala em $GOPATH/bin)
go install github.com/flavio/germanio@latest

# Verificar instalação
germanio version
```

### 2.2 Criar Novo Projeto

```bash
# Cria estrutura básica do projeto
germanio new minha-loja

# Estrutura criada:
# minha-loja/
#   inicio.ge

# Navegar para o diretório
cd minha-loja

# Executar
germanio run inicio.ge
```

### 2.3 Inicializar Projeto Completo

O comando `init` cria um projeto com `.env`, `.gitignore` e `Dockerfile` prontos:

```bash
germanio init minha-loja

# Estrutura criada:
# minha-loja/
#   inicio.ge
#   .env
#   .gitignore
#   Dockerfile

cd minha-loja
germanio run inicio.ge
```

### 2.4 Comandos CLI

```bash
# Executar aplicação (porta padrão: 8080)
germanio run inicio.ge

# Executar em porta personalizada
germanio run inicio.ge 3000

# Atalho — sem subcomando "run"
germanio inicio.ge
germanio inicio.ge 3000

# Verificar sintaxe sem executar
germanio check inicio.ge

# Criar novo projeto
germanio new nome-do-projeto

# Inicializar projeto com arquivos extras
germanio init nome-do-projeto

# Gerar Dockerfile no diretório atual
germanio docker

# Ver versão
germanio version

# Ajuda
germanio help
```

### 2.5 Desenvolvimento com Auto-reload

Germanio não tem hot-reload nativo. Use ferramentas externas:

```bash
# Usando 'air' (https://github.com/air-verse/air)
go install github.com/air-verse/air@latest
air -- run inicio.ge

# Usando 'watchexec'
watchexec -e fg -- germanio run inicio.ge

# Usando 'nodemon' (requer Node.js)
npx nodemon --watch "*.ge" --exec "germanio run inicio.ge"
```

### 2.6 Verificar Antes de Deploy

```bash
# Verifica toda a sintaxe sem iniciar o servidor
germanio check inicio.ge

# Saída de sucesso:
# [germanio] Sintaxe OK: inicio.ge
```

---

## 3. Variáveis de Ambiente

Crie um arquivo `.env` na mesma pasta do `.ge` para configurar o ambiente.

### 3.1 Arquivo `.env` Padrão

```env
# Porta do servidor
GERMANIO_PORT=8080

# Banco de dados
GERMANIO_DB_TYPE=sqlite
GERMANIO_DB_NAME=minha_loja.db

# Para PostgreSQL:
# GERMANIO_DB_TYPE=postgres
# GERMANIO_DB_HOST=localhost
# GERMANIO_DB_PORT=5432
# GERMANIO_DB_NAME=minha_loja
# GERMANIO_DB_USER=postgres
# GERMANIO_DB_PASS=senha_secreta

# Para MySQL:
# GERMANIO_DB_TYPE=mysql
# GERMANIO_DB_HOST=localhost
# GERMANIO_DB_PORT=3306
# GERMANIO_DB_NAME=minha_loja
# GERMANIO_DB_USER=root
# GERMANIO_DB_PASS=senha_secreta

# Autenticação JWT (gere um valor aleatório longo!)
GERMANIO_JWT_SECRET=mude-isso-para-um-segredo-muito-longo-e-aleatorio

# E-mail (SMTP)
GERMANIO_SMTP_HOST=smtp.gmail.com
GERMANIO_SMTP_PORT=587
GERMANIO_SMTP_USER=sistema@empresa.com
GERMANIO_SMTP_PASS=app_password_aqui
GERMANIO_SMTP_FROM=Sistema <sistema@empresa.com>
```

### 3.2 `.gitignore` Recomendado

```gitignore
# Banco de dados SQLite
*.db
*.db-shm
*.db-wal

# Variáveis de ambiente (nunca commitar!)
.env

# Binários
germanio
germanio.exe

# Uploads
uploads/

# Logs
*.log
```

### 3.3 Configuração via Arquivo `.ge`

Alternativamente, configure diretamente no arquivo `.ge`:

```germanio
banco
  driver: postgres
  host: "db.exemplo.com"
  porta: "5432"
  nome: "producao"
  usuario: "app_user"
  senha: "senha_super_segura"
```

**Recomendação:** Use variáveis de ambiente em produção — nunca commite credenciais no código.

---

## 4. SQLite — Configuração Padrão

SQLite é o banco padrão e requer **zero configuração** para começar.

### 4.1 Configuração Mínima

```germanio
sistema minha-loja
# sem bloco 'banco' = SQLite automático
```

O arquivo de banco é criado como `<nome-sistema>.db` no diretório de trabalho.

### 4.2 Configuração Explícita

```germanio
banco
  driver: sqlite
  nome: "dados/minha_loja.db"
```

### 4.3 Características do SQLite no Germanio

- **WAL Mode** habilitado automaticamente (`journal_mode=WAL`) para melhor concorrência de leitura
- **Foreign Keys** habilitadas automaticamente (`foreign_keys=ON`)
- **Sem servidor** — arquivo local, zero dependências externas
- Adequado para: desenvolvimento, pequenos deploys, apps com até ~10k usuários simultâneos

### 4.4 Migração de SQLite para PostgreSQL

```bash
# 1. Exportar dados
curl http://localhost:8080/api/produto/export/json > produto.json
curl http://localhost:8080/api/cliente/export/json > cliente.json
# (repetir para cada modelo)

# 2. Atualizar configuração do banco no .ge

# 3. Reiniciar (cria tabelas automaticamente)
germanio run inicio.ge

# 4. Importar dados via API
cat produto.json | jq -c '.[]' | while read item; do
  curl -X POST http://localhost:8080/api/produto \
    -H "Content-Type: application/json" \
    -d "$item"
done
```

---

## 5. PostgreSQL — Produção

### 5.1 Instalar PostgreSQL

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install -y postgresql postgresql-contrib

# Iniciar serviço
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**Docker (desenvolvimento rápido):**
```bash
docker run -d \
  --name postgres-germanio \
  -e POSTGRES_USER=germanio \
  -e POSTGRES_PASSWORD=germaniopass \
  -e POSTGRES_DB=minha_loja \
  -p 5432:5432 \
  postgres:16-alpine
```

### 5.2 Criar Banco e Usuário

```sql
-- Como usuário postgres
sudo -u postgres psql

-- Criar usuário da aplicação
CREATE USER app_user WITH PASSWORD 'senha_super_segura';

-- Criar banco
CREATE DATABASE minha_loja OWNER app_user;

-- Conceder privilégios
GRANT ALL PRIVILEGES ON DATABASE minha_loja TO app_user;

-- Sair
\q
```

### 5.3 Configuração no `.ge`

```germanio
banco
  driver: postgres
  host: "localhost"
  porta: "5432"
  nome: "minha_loja"
  usuario: "app_user"
  senha: "senha_super_segura"
```

### 5.4 String de Conexão (DSN)

O Germanio monta internamente:
```
host=localhost port=5432 user=app_user password=senha_super_segura dbname=minha_loja sslmode=disable
```

### 5.5 SSL em Produção (PostgreSQL)

Para habilitar SSL, configure no PostgreSQL e ajuste a DSN manualmente (ou use um tunel seguro como VPN/bastion).

### 5.6 Connection Pooling com PgBouncer

Para alta carga, use PgBouncer entre o Germanio e o PostgreSQL:

```ini
# /etc/pgbouncer/pgbouncer.ini
[databases]
minha_loja = host=127.0.0.1 port=5432 dbname=minha_loja

[pgbouncer]
listen_port = 5433
listen_addr = 127.0.0.1
auth_type = md5
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
```

No `.ge`:
```germanio
banco
  driver: postgres
  host: "127.0.0.1"
  porta: "5433"  # porta do PgBouncer
  nome: "minha_loja"
  usuario: "app_user"
  senha: "senha_super_segura"
```

---

## 6. MySQL — Alternativa

### 6.1 Instalar MySQL

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install -y mysql-server
sudo systemctl start mysql
sudo systemctl enable mysql

# Configuração inicial de segurança
sudo mysql_secure_installation
```

### 6.2 Criar Banco e Usuário

```sql
sudo mysql -u root -p

CREATE DATABASE minha_loja CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'app_user'@'localhost' IDENTIFIED BY 'senha_super_segura';
GRANT ALL PRIVILEGES ON minha_loja.* TO 'app_user'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

### 6.3 Configuração no `.ge`

```germanio
banco
  driver: mysql
  host: "localhost"
  porta: "3306"
  nome: "minha_loja"
  usuario: "app_user"
  senha: "senha_super_segura"
```

### 6.4 String de Conexão (DSN)

O Germanio monta internamente:
```
app_user:senha_super_segura@tcp(localhost:3306)/minha_loja?parseTime=true&charset=utf8mb4
```

### 6.5 Diferenças MySQL no Germanio

| Comportamento | SQLite | MySQL |
|---|---|---|
| Auto-increment | `AUTOINCREMENT` | `AUTO_INCREMENT` |
| Tipo texto | `TEXT` | `VARCHAR(500)` |
| Tipo data | `DATETIME` | `DATETIME` |
| Placeholder | `?` | `?` |

---

## 7. Docker

### 7.1 Gerar Dockerfile Automaticamente

```bash
# No diretório do projeto
germanio docker

# Conteúdo gerado:
```

```dockerfile
# Generated by germanio docker
FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o germanio .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /build/germanio /usr/local/bin/germanio
COPY *.ge ./

EXPOSE 8080
CMD ["germanio", "run", "inicio.ge"]
```

### 7.2 Dockerfile Personalizado para Produção

```dockerfile
# ===== Stage 1: Build =====
FROM golang:1.21-alpine AS builder

# Instalar dependências do sistema
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache de dependências Go
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar binário estático otimizado
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -extldflags=-static" \
    -o germanio .

# ===== Stage 2: Runtime =====
FROM scratch

# Certificados SSL para chamadas HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Timezone
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Binário compilado
COPY --from=builder /build/germanio /germanio

# Arquivos da aplicação
COPY *.ge /app/

WORKDIR /app

# Volume para dados persistentes (SQLite, uploads)
VOLUME ["/app/data", "/app/uploads"]

EXPOSE 8080

ENV GERMANIO_PORT=8080

ENTRYPOINT ["/germanio"]
CMD ["run", "inicio.ge"]
```

### 7.3 Construir e Executar

```bash
# Construir imagem
docker build -t minha-loja:latest .

# Executar (SQLite)
docker run -d \
  --name minha-loja \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/uploads:/app/uploads \
  minha-loja:latest

# Verificar status
docker logs minha-loja
docker ps

# Parar
docker stop minha-loja
docker rm minha-loja
```

### 7.4 Com PostgreSQL externo

```bash
docker run -d \
  --name minha-loja \
  -p 8080:8080 \
  -e GERMANIO_DB_TYPE=postgres \
  -e GERMANIO_DB_HOST=postgres.exemplo.com \
  -e GERMANIO_DB_NAME=minha_loja \
  -e GERMANIO_DB_USER=app_user \
  -e GERMANIO_DB_PASS=senha_segura \
  -v $(pwd)/uploads:/app/uploads \
  minha-loja:latest
```

---

## 8. Docker Compose

### 8.1 docker-compose.yml — SQLite (simples)

```yaml
version: '3.9'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - app_data:/app/data
      - app_uploads:/app/uploads
    environment:
      - GERMANIO_PORT=8080
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  app_data:
  app_uploads:
```

### 8.2 docker-compose.yml — PostgreSQL (produção)

```yaml
version: '3.9'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - app_uploads:/app/uploads
    environment:
      - GERMANIO_PORT=8080
      - GERMANIO_DB_TYPE=postgres
      - GERMANIO_DB_HOST=db
      - GERMANIO_DB_PORT=5432
      - GERMANIO_DB_NAME=minha_loja
      - GERMANIO_DB_USER=germanio_user
      - GERMANIO_DB_PASS=${DB_PASSWORD}
      - GERMANIO_JWT_SECRET=${JWT_SECRET}
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - app_network
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 15s

  db:
    image: postgres:16-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=minha_loja
      - POSTGRES_USER=germanio_user
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    restart: unless-stopped
    networks:
      - app_network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U germanio_user -d minha_loja"]
      interval: 10s
      timeout: 5s
      retries: 5

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - app
    restart: unless-stopped
    networks:
      - app_network

networks:
  app_network:
    driver: bridge

volumes:
  postgres_data:
  app_uploads:
```

**Arquivo `.env` para o compose:**
```env
DB_PASSWORD=senha_super_segura_aleatoria
JWT_SECRET=jwt_secret_muito_longo_e_aleatorio_aqui
```

### 8.3 Comandos Docker Compose

```bash
# Iniciar em background
docker compose up -d

# Ver logs
docker compose logs -f app

# Ver logs de todos os serviços
docker compose logs -f

# Parar todos os serviços
docker compose down

# Parar e remover volumes (CUIDADO: apaga dados)
docker compose down -v

# Reconstruir após mudanças no código
docker compose up -d --build

# Escalar instâncias do app (requer load balancer)
docker compose up -d --scale app=3

# Status dos serviços
docker compose ps
```

---

## 9. HTTPS com Nginx (Reverse Proxy)

### 9.1 Instalar Nginx e Certbot

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx

# CentOS/RHEL
sudo dnf install -y nginx certbot python3-certbot-nginx
```

### 9.2 Configuração Nginx (HTTP → HTTPS)

```nginx
# /etc/nginx/sites-available/minha-loja
server {
    listen 80;
    listen [::]:80;
    server_name minha-loja.com.br www.minha-loja.com.br;

    # Redirecionar todo HTTP para HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name minha-loja.com.br www.minha-loja.com.br;

    # Certificados SSL (gerados pelo Certbot)
    ssl_certificate /etc/letsencrypt/live/minha-loja.com.br/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/minha-loja.com.br/privkey.pem;

    # Configurações SSL modernas
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;

    # Headers de segurança
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;

    # Limite de upload (deve ser >= limite do Germanio: 32MB)
    client_max_body_size 35M;

    # Logs
    access_log /var/log/nginx/minha-loja.access.log;
    error_log /var/log/nginx/minha-loja.error.log;

    # Proxy para o Germanio
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;

        # Headers necessários para WebSocket
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Headers padrão de proxy
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Buffer
        proxy_buffering on;
        proxy_buffer_size 8k;
        proxy_buffers 16 8k;
    }

    # WebSocket — caminho dedicado
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400s;  # 24h — conexão persistente
    }
}
```

### 9.3 Ativar Configuração

```bash
# Habilitar site
sudo ln -s /etc/nginx/sites-available/minha-loja /etc/nginx/sites-enabled/

# Verificar sintaxe
sudo nginx -t

# Recarregar Nginx
sudo systemctl reload nginx
```

### 9.4 Certificado SSL com Let's Encrypt (gratuito)

```bash
# Obter certificado
sudo certbot --nginx -d minha-loja.com.br -d www.minha-loja.com.br

# Testar renovação automática
sudo certbot renew --dry-run

# Renovação automática já é configurada pelo certbot via cron/systemd timer
# Verificar: sudo systemctl status certbot.timer
```

---

## 10. Systemd — Serviço Linux

Configure o Germanio como um serviço do sistema para reinicialização automática.

### 10.1 Preparar Estrutura de Diretórios

```bash
# Criar usuário dedicado (sem shell, sem home login)
sudo useradd -r -s /bin/false -d /opt/minha-loja germanio

# Criar diretório da aplicação
sudo mkdir -p /opt/minha-loja
sudo mkdir -p /opt/minha-loja/uploads

# Copiar binário e arquivos
sudo cp germanio /usr/local/bin/germanio
sudo chmod +x /usr/local/bin/germanio
sudo cp inicio.ge /opt/minha-loja/
sudo cp .env /opt/minha-loja/

# Definir permissões
sudo chown -R germanio:germanio /opt/minha-loja
sudo chmod 750 /opt/minha-loja
```

### 10.2 Arquivo de Serviço Systemd

```ini
# /etc/systemd/system/minha-loja.service

[Unit]
Description=Minha Loja — Germanio Application
Documentation=https://github.com/flavio/germanio
After=network.target
Wants=network-online.target

# Se usar PostgreSQL, descomente:
# After=postgresql.service
# Requires=postgresql.service

[Service]
Type=simple
User=germanio
Group=germanio
WorkingDirectory=/opt/minha-loja

# Comando de inicialização
ExecStart=/usr/local/bin/germanio run inicio.ge 8080

# Reiniciar automaticamente em caso de falha
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=60s
StartLimitBurst=3

# Variáveis de ambiente
EnvironmentFile=/opt/minha-loja/.env

# Saída de logs
StandardOutput=journal
StandardError=journal
SyslogIdentifier=minha-loja

# Segurança — limitar privilégios
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/minha-loja
PrivateTmp=true

# Limite de arquivos abertos
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

### 10.3 Gerenciar o Serviço

```bash
# Recarregar configurações do systemd
sudo systemctl daemon-reload

# Habilitar início automático no boot
sudo systemctl enable minha-loja

# Iniciar serviço
sudo systemctl start minha-loja

# Verificar status
sudo systemctl status minha-loja

# Ver logs em tempo real
sudo journalctl -u minha-loja -f

# Ver últimas 100 linhas de log
sudo journalctl -u minha-loja -n 100

# Reiniciar
sudo systemctl restart minha-loja

# Parar
sudo systemctl stop minha-loja

# Desabilitar início automático
sudo systemctl disable minha-loja
```

### 10.4 Script de Deploy

```bash
#!/bin/bash
# /opt/deploy-minha-loja.sh

set -e

APP_DIR="/opt/minha-loja"
SERVICE="minha-loja"

echo "=== Deploy Minha Loja ==="

# 1. Parar serviço
sudo systemctl stop $SERVICE || true

# 2. Fazer backup do banco (SQLite)
if [ -f "$APP_DIR/minha_loja.db" ]; then
    cp "$APP_DIR/minha_loja.db" "$APP_DIR/backup_$(date +%Y%m%d_%H%M%S).db"
    echo "Backup criado"
fi

# 3. Atualizar binário
sudo cp germanio /usr/local/bin/germanio
sudo chmod +x /usr/local/bin/germanio

# 4. Atualizar arquivos .ge
sudo cp *.ge "$APP_DIR/"
sudo chown germanio:germanio "$APP_DIR/"*.ge

# 5. Reiniciar serviço
sudo systemctl start $SERVICE

# 6. Verificar saúde
sleep 3
if curl -sf http://localhost:8080/health > /dev/null; then
    echo "Deploy concluído com sucesso!"
else
    echo "ERRO: serviço não está respondendo!"
    sudo journalctl -u $SERVICE -n 20
    exit 1
fi
```

---

## 11. Performance e Tuning

### 11.1 SQLite — Otimizações

O Germanio já habilita WAL mode automaticamente. Configurações adicionais para alta carga:

```germanio
banco
  driver: sqlite
  nome: "dados/app.db"
```

**Configurações WAL já aplicadas pelo Germanio:**
```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
```

**Configurações adicionais recomendadas para produção (via SQL direto):**
```sql
PRAGMA cache_size = -64000;    -- 64MB de cache
PRAGMA synchronous = NORMAL;   -- mais rápido que FULL, seguro com WAL
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 268435456;  -- 256MB memory-mapped I/O
```

### 11.2 PostgreSQL — Otimizações

**`postgresql.conf` recomendado para servidor com 4GB RAM:**

```ini
# Memória
shared_buffers = 1GB            # 25% da RAM
effective_cache_size = 3GB      # 75% da RAM
work_mem = 64MB
maintenance_work_mem = 256MB

# Write-Ahead Log
wal_buffers = 16MB
checkpoint_completion_target = 0.9
wal_level = replica

# Conexões
max_connections = 200

# Performance de queries
random_page_cost = 1.1          # Para SSD
effective_io_concurrency = 200  # Para SSD
default_statistics_target = 100

# Logging
log_min_duration_statement = 1000  # Log queries > 1s
```

### 11.3 Nginx — Otimizações

```nginx
# /etc/nginx/nginx.conf

worker_processes auto;
worker_rlimit_nofile 65536;

events {
    worker_connections 4096;
    use epoll;
    multi_accept on;
}

http {
    # Compressão
    gzip on;
    gzip_vary on;
    gzip_types application/json text/plain application/javascript text/css;
    gzip_min_length 1000;

    # Cache de conexões
    keepalive_timeout 65;
    keepalive_requests 1000;

    # Buffers
    client_body_buffer_size 128k;
    proxy_buffer_size 8k;
    proxy_buffers 8 32k;
    proxy_busy_buffers_size 64k;

    # Rate limiting (opcional)
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;

    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites-enabled/*;
}
```

### 11.4 Go Runtime — Variáveis de Ambiente

```bash
# Ajustar uso de CPU (padrão já é automático)
export GOMAXPROCS=$(nproc)

# Ajustar GC (menos pauses, mais memória)
export GOGC=200

# Executar com configurações
GOMAXPROCS=$(nproc) GOGC=200 germanio run inicio.ge
```

### 11.5 Capacidade Estimada

| Banco | Usuários Simultâneos | Registros | Cenário |
|---|---|---|---|
| SQLite (WAL) | ~100 | até 1M | Pequenas apps, startups |
| PostgreSQL | ~5.000 | ilimitado | Apps médias e grandes |
| MySQL | ~3.000 | ilimitado | Apps médias |
| PG + PgBouncer | ~20.000 | ilimitado | Alta demanda |

---

## 12. Estratégias de Backup

### 12.1 Backup de SQLite

SQLite usa WAL mode — para backup consistente sem parar o servidor:

```bash
#!/bin/bash
# /usr/local/bin/backup-sqlite.sh

APP_DIR="/opt/minha-loja"
BACKUP_DIR="/var/backups/minha-loja"
DB_FILE="$APP_DIR/minha_loja.db"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Backup online usando o comando SQLite (consistente mesmo com WAL)
sqlite3 "$DB_FILE" ".backup $BACKUP_DIR/backup_$DATE.db"

# Ou copiar com sync forçado (menos seguro)
# cp "$DB_FILE" "$BACKUP_DIR/backup_$DATE.db"

# Comprimir
gzip "$BACKUP_DIR/backup_$DATE.db"

# Manter apenas últimos 30 backups
ls -t "$BACKUP_DIR"/*.db.gz | tail -n +31 | xargs rm -f

echo "Backup concluído: $BACKUP_DIR/backup_$DATE.db.gz"
```

**Agendar backup automático (cron):**
```bash
# crontab -e
# Backup diário às 2h da manhã
0 2 * * * /usr/local/bin/backup-sqlite.sh >> /var/log/backup-germanio.log 2>&1
```

### 12.2 Backup de PostgreSQL

```bash
#!/bin/bash
# /usr/local/bin/backup-postgres.sh

DB_NAME="minha_loja"
DB_USER="germanio_user"
BACKUP_DIR="/var/backups/minha-loja"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Dump completo com compressão
PGPASSWORD="$DB_PASSWORD" pg_dump \
  -h localhost \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  -F c \  # formato custom (comprimido)
  -f "$BACKUP_DIR/backup_$DATE.dump"

# Backup apenas estrutura
PGPASSWORD="$DB_PASSWORD" pg_dump \
  -h localhost \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  --schema-only \
  -f "$BACKUP_DIR/schema_$DATE.sql"

# Manter apenas últimos 30 dumps
ls -t "$BACKUP_DIR"/*.dump | tail -n +31 | xargs rm -f

echo "Backup PostgreSQL concluído: $BACKUP_DIR/backup_$DATE.dump"
```

**Restaurar backup PostgreSQL:**
```bash
pg_restore \
  -h localhost \
  -U germanio_user \
  -d minha_loja \
  --clean \
  backup_20260409_020000.dump
```

### 12.3 Backup de Uploads

```bash
#!/bin/bash
# Sincronizar uploads para armazenamento externo (S3, Backblaze, etc.)

# Para AWS S3
aws s3 sync /opt/minha-loja/uploads/ s3://meu-bucket/minha-loja-uploads/

# Para Backblaze B2
b2 sync /opt/minha-loja/uploads/ b2://meu-bucket/uploads/

# Backup local comprimido
tar -czf /var/backups/uploads_$(date +%Y%m%d).tar.gz /opt/minha-loja/uploads/
```

### 12.4 Estratégia de Retenção Recomendada

| Período | Frequência | Destino |
|---|---|---|
| Diário | Backup completo | Local (30 dias) |
| Semanal | Backup completo | S3/Cloud (12 semanas) |
| Mensal | Backup completo | Frio/Archive (12 meses) |
| Uploads | Sincronização contínua | S3/Cloud |

---

## 13. Monitoramento

### 13.1 Health Check Básico

```bash
# Script de verificação
#!/bin/bash
RESPONSE=$(curl -sf http://localhost:8080/health)
if [ "$RESPONSE" = '{"status":"ok"}' ]; then
    echo "OK: Germanio está respondendo"
else
    echo "ERRO: Germanio não está respondendo!"
    # Enviar alerta (email, Slack, PagerDuty, etc.)
    systemctl restart minha-loja
fi
```

### 13.2 Monitoramento com `/api/_stats`

```bash
# Verificar contagem de registros
curl http://localhost:8080/api/_stats | jq .

# Alertar se pedidos pendentes > 100
PENDENTES=$(curl -s http://localhost:8080/api/_stats | jq '.pedido.statuses.pendente // 0')
if [ "$PENDENTES" -gt 100 ]; then
    echo "ALERTA: $PENDENTES pedidos pendentes!"
fi
```

### 13.3 Logs do Sistema

```bash
# Logs do serviço systemd (tempo real)
journalctl -u minha-loja -f

# Últimas 100 linhas
journalctl -u minha-loja -n 100

# Logs de hoje
journalctl -u minha-loja --since today

# Logs com erros apenas
journalctl -u minha-loja -p err

# Salvar logs em arquivo
journalctl -u minha-loja --since "2026-04-01" > logs_abril.txt
```

### 13.4 Integração com Prometheus (via exportador externo)

```bash
# Configurar um cron que expõe métricas para Prometheus
#!/bin/bash
STATS=$(curl -s http://localhost:8080/api/_stats)
echo "# HELP germanio_model_count Total de registros por modelo"
echo "# TYPE germanio_model_count gauge"
echo "$STATS" | jq -r 'to_entries[] | "germanio_model_count{model=\"\(.key)\"} \(.value.count)"'
```

---

## 14. Segurança em Produção

### 14.1 Checklist de Segurança

- [ ] **JWT Secret** forte (mínimo 32 caracteres aleatórios)
- [ ] Nunca commitar `.env` no Git
- [ ] Usar HTTPS (TLS 1.2+) em produção
- [ ] Banco de dados em rede privada (não exposto externamente)
- [ ] Usuário do sistema sem privilégios de root
- [ ] Firewall: apenas portas 80 e 443 públicas
- [ ] Backups automáticos e testados
- [ ] Logs monitorados

### 14.2 Gerar JWT Secret Seguro

```bash
# Linux/Mac — 64 bytes aleatórios em base64
openssl rand -base64 64

# Ou usando Python
python3 -c "import secrets; print(secrets.token_hex(64))"

# Resultado (exemplo):
# 8f3b9c1d4e7a2f5b0e8c3d6f9a2b5e8c1d4f7a0b3e6c9f2a5b8d1e4f7a0c3d6f
```

### 14.3 Firewall com UFW

```bash
# Instalar e configurar UFW
sudo apt install ufw

# Permitir SSH
sudo ufw allow ssh

# Permitir HTTP e HTTPS apenas
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Bloquear porta do Germanio diretamente (só via Nginx)
sudo ufw deny 8080/tcp

# Ativar
sudo ufw enable
sudo ufw status
```

### 14.4 Fail2Ban para Proteção contra Brute Force

```bash
sudo apt install fail2ban

# /etc/fail2ban/jail.local
[nginx-http-auth]
enabled = true
port = http,https
logpath = /var/log/nginx/minha-loja.access.log
maxretry = 5
bantime = 3600

sudo systemctl restart fail2ban
```

### 14.5 Senhas Seguras por Padrão

O Germanio usa **bcrypt** (custo 10) para hashing de senhas — padrão da indústria. Não é necessária nenhuma configuração adicional.

---

## 15. Checklist de Deploy

### Pré-Deploy

- [ ] `germanio check inicio.ge` passou sem erros
- [ ] Banco de dados configurado e acessível
- [ ] Variáveis de ambiente definidas no `.env`
- [ ] JWT Secret definido e seguro
- [ ] Backup do banco de dados feito
- [ ] Uploads anteriores preservados

### Deploy

- [ ] Binário `germanio` compilado para o SO alvo
- [ ] Arquivos `.ge` transferidos para o servidor
- [ ] Permissões de arquivo corretas
- [ ] Serviço systemd configurado e ativo
- [ ] Nginx configurado com SSL

### Pós-Deploy

- [ ] `GET /health` retorna `{"status":"ok"}`
- [ ] Login de teste funcionando
- [ ] `GET /api/_stats` retorna dados corretos
- [ ] WebSocket conectando (`/ws`)
- [ ] Uploads funcionando (`POST /upload`)
- [ ] Logs sem erros (`journalctl -u minha-loja -n 50`)
- [ ] Backup automático agendado
- [ ] Monitoramento ativo

---

## Referência Rápida de Comandos

```bash
# ===== CLI =====
germanio run inicio.ge          # Executar na porta 8080
germanio run inicio.ge 3000     # Executar na porta 3000
germanio inicio.ge              # Atalho para run
germanio check inicio.ge        # Verificar sintaxe
germanio new meu-app            # Novo projeto básico
germanio init meu-app           # Novo projeto completo
germanio docker                 # Gerar Dockerfile
germanio version                # Ver versão

# ===== Docker =====
docker build -t app .                    # Construir imagem
docker run -p 8080:8080 app             # Executar container
docker compose up -d                     # Compose em background
docker compose logs -f app              # Ver logs
docker compose down                     # Parar

# ===== Systemd =====
sudo systemctl start minha-loja         # Iniciar
sudo systemctl stop minha-loja          # Parar
sudo systemctl restart minha-loja       # Reiniciar
sudo systemctl status minha-loja        # Status
sudo journalctl -u minha-loja -f       # Logs em tempo real

# ===== Backup SQLite =====
sqlite3 app.db ".backup backup.db"     # Backup online
gzip backup.db                          # Comprimir

# ===== Backup PostgreSQL =====
pg_dump -U user -d dbname -F c -f backup.dump   # Backup
pg_restore -U user -d dbname backup.dump        # Restaurar
```
