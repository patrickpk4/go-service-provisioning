# Projeto Korp

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![Ansible](https://img.shields.io/badge/Ansible-Automation-EE0000?logo=ansible&logoColor=white)](https://docs.ansible.com/)
[![Prometheus](https://img.shields.io/badge/Prometheus-Monitoring-E6522C?logo=prometheus&logoColor=white)](https://prometheus.io/)
[![Grafana](https://img.shields.io/badge/Grafana-Dashboard-F46800?logo=grafana&logoColor=white)](https://grafana.com/)

Serviço HTTP em Go, executado em containers Docker e acessível por meio de um
proxy reverso Nginx. O projeto inclui monitoramento com Prometheus e Grafana e
provisionamento automatizado com Ansible.

> **Documentação técnica:** [guia completo de decisões técnicas em PDF](docs/decisoes-tecnicas.pdf) | [versão Markdown](docs/decisoes-tecnicas.md) | [defesa técnica para entrevista](docs/defesa-tecnica-entrevista.md)

## Visão geral

```text
Cliente -> Nginx :80 -> rede Docker bridge -> http-server-projeto-korp :8080
                       |                     |
                       |                     +--> Prometheus :9090
                       v                           |
                 nginx-exporter :9113 ------------+
                                                   v
                                            Grafana :3000
```

O serviço Go, o Nginx, o exporter, o Prometheus e o Grafana se comunicam pela
rede `korp-network`. A aplicação Go não publica a porta 8080 diretamente no
host.

## Componentes

| Componente | Responsabilidade |
| --- | --- |
| Go | Serviço HTTP e métricas |
| Docker | Empacotamento e execução |
| Docker Compose | Orquestração dos containers |
| Nginx | Proxy reverso na porta 80 |
| Nginx Prometheus Exporter | Converte `stub_status` do Nginx em métricas |
| Prometheus | Coleta e armazenamento de métricas |
| Grafana | Visualização das métricas |
| Ansible | Provisionamento do host Linux |

## Pré-requisitos

Execução local com Docker Compose:

- Docker Engine;
- Docker Compose v2;
- `curl`.

Execução com Ansible:

- host Linux com Docker instalado ou com gerenciador de pacotes suportado;
- Ansible instalado;
- usuário com acesso administrativo via `sudo`.

## Execução local

```bash
docker compose up -d --build
docker compose ps
curl http://localhost:80/projeto-korp
```

Resposta esperada:

```json
{"horario":"03/09/2026 18:07:35","nome":"Projeto Korp"}
```

O campo `horario` é calculado dinamicamente em UTC a cada requisição.

## Monitoramento

| Recurso | URL |
| --- | --- |
| Aplicação | http://localhost:80/projeto-korp |
| Health check | http://localhost:80/health |
| Métricas | http://localhost:80/metrics |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |

Credenciais locais do Grafana: `admin` / `admin`.

O dashboard `Projeto Korp - Monitoramento` é carregado automaticamente e
apresenta a disponibilidade da aplicação e do Nginx, total de requisições, taxa
de requisições, latência p95, erros HTTP 5xx e histórico de disponibilidade.

Métricas monitoradas:

- `up{job="http-server-projeto-korp"}`: disponibilidade real da aplicação Go;
- `nginx_up`: disponibilidade real do Nginx, medida pelo exporter;
- `http_requests_total`: contador de requisições;
- `http_responses_total{status="..."}`: contador por status HTTP;
- `http_request_duration_seconds`: histograma de duração.


## Provisionamento com Ansible

O playbook [ansible/site.yml](ansible/site.yml) automatiza o ambiente completo
em um host Ubuntu: instala Python, Docker e Compose, cria a rede bridge,
copia os arquivos, constrói a imagem, configura Nginx, Prometheus e Grafana,
inicia a stack, valida o serviço e exibe no console a resposta JSON do endpoint
`/projeto-korp`.

Para executar na máquina Ubuntu local, instale o Ansible e crie o inventário:

```bash
sudo apt update
sudo apt install -y ansible
cp ansible/inventory.ini.example ansible/inventory.ini
```

Troque `SEU_USUARIO` pelo seu usuário Linux. Depois, este é o comando único que
instala e configura toda a stack:

```bash
ansible-playbook \
	-i ansible/inventory.ini \
	ansible/site.yml \
	-K
```

O parâmetro `-K` solicita a senha do `sudo` e permite que o playbook instale
Docker, Compose e Python, crie a rede, faça o build da imagem, suba os
containers e execute as validações finais. A instalação do Ansible e a criação
do inventário são apenas preparação inicial; o provisionamento completo ocorre
com o comando acima.

O playbook instala Docker automaticamente em Debian/Ubuntu e Red Hat-like.
Para um container Linux ou outra distribuição, o Docker precisa estar
disponível antecipadamente. Em um container, monte o socket Docker do host e
execute com permissões suficientes; um container Linux comum, sem daemon ou
socket Docker, não pode criar os demais containers.

Quando o Docker já estiver disponível, defina `install_docker=false` e
`manage_docker_service=false` para o host no inventário. Para um host remoto,
substitua `localhost` pelo endereço e informe a chave SSH com `--private-key`.

## Testes e validações

```bash
go test ./... -race -cover
docker compose config
python3 -m json.tool grafana/dashboards/projeto-korp.json
```

## Estrutura

```text
.
├── ansible/                         # Inventário e playbook
├── grafana/                         # Datasource e dashboard
├── nginx/                           # Proxy reverso
├── prometheus/                      # Coleta de métricas
├── Dockerfile                       # Imagem da aplicação Go
├── docker-compose.yml               # Stack de containers
├── go.mod                            # Módulo Go
├── main.go                           # Servidor e métricas
└── main_test.go                      # Testes unitários
```

## Decisões técnicas

- A imagem usa build multi-stage e uma imagem final `scratch`.
- A porta 8080 fica restrita à rede Docker; o Nginx é o ponto de entrada.
- O dashboard é provisionado por arquivos e permanece em JSON válido.
- Os comentários dos arquivos de configuração usam `#`. Os arquivos Go não
  possuem comentários, pois `#` não é aceito pela linguagem Go.

## Limpeza

```bash
docker compose down
docker compose down --volumes
```
