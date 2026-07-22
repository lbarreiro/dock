# Arquitetura do Dock

## Objetivo

O Dock é uma consola moderna, minimalista e extremamente leve para administração de servidores Linux.

O projeto foi concebido para funcionar continuamente em equipamentos de baixo consumo, como Raspberry Pi, mantendo uma utilização mínima de CPU e memória.

## Princípios

- Um único binário.
- Arquitetura modular.
- Interface única.
- Módulos independentes.
- Baixo consumo de recursos.
- Configuração simples.
- Código fácil de manter.

## Estrutura

cmd/
    Ponto de entrada da aplicação.

internal/server/
    Servidor HTTP.

internal/modules/
    Implementação dos módulos.

internal/config/
    Leitura da configuração.

web/
    Interface do utilizador.

docs/
    Documentação técnica.

## Módulos

Cada módulo tem uma única responsabilidade.

Os módulos nunca comunicam diretamente entre si.

Toda a comunicação passa pelo servidor.

Os módulos fornecem apenas dados.

A interface é responsável por apresentar esses dados.

