# App

## Objetivo

O pacote `app` é o ponto central da aplicação.

É responsável por inicializar todos os componentes do Dock e gerir o ciclo de vida da aplicação.

Nenhum componente conhece os detalhes de inicialização dos restantes.

## Responsabilidades

- Carregar a configuração.
- Inicializar o cliente Docker.
- Criar o servidor HTTP.
- Registar todos os módulos.
- Arrancar a aplicação.

## Fluxo de inicialização

main.go
    ↓
app.Run()
    ↓
Load Config
    ↓
Create Docker Client
    ↓
Create HTTP Server
    ↓
Register Modules
    ↓
Start HTTP Server

## Princípios

- O `main.go` deve conter o mínimo de código possível.
- Não utilizar variáveis globais.
- Todas as dependências são criadas no `app`.
- Os componentes recebem apenas as dependências de que necessitam.
- Os módulos nunca comunicam diretamente entre si.

## Objetivo futuro

O pacote `app` deverá tornar simples adicionar novos componentes, como:

- Autenticação
- WebSockets
- Scheduler
- Sistema de permissões
- Plugins

Sem alterar a estrutura principal da aplicação.
