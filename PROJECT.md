# Dock Project Constitution

> Este documento define as regras fundamentais do projeto.
> Qualquer alteração a estas regras deve ser uma decisão consciente.

---

# Missão

O Dock é uma consola moderna, minimalista e extremamente leve para administração de servidores Linux.

O objetivo é permitir executar ações e visualizar o estado do sistema através de um painel configurável, rápido e intuitivo.

---

# Filosofia

Leve.
Rápido.
Seguro.
Elegante.

Nenhuma funcionalidade deve comprometer estes princípios.

---

# Arquitetura

- A estrutura de diretórios é considerada estável.
- Não criar novas pastas sem uma razão arquitetónica.
- Antes de escrever código, definir a arquitetura.
- A simplicidade vence a complexidade.

---

# Desenvolvimento

- Um passo de cada vez.
- Confirmar cada passo antes de avançar.
- Criar ficheiros utilizando sempre `tee`.
- Não utilizar editores interativos durante o desenvolvimento.
- Todo o código novo deve ser reutilizável.

---

# Módulos

Todos os módulos seguem a mesma estrutura.

module.go
service.go
api.go
models.go
README.md

Regras:

- Os módulos são independentes.
- Os módulos nunca comunicam diretamente entre si.
- Os módulos apenas fornecem dados.
- A interface é responsável pela apresentação.

---

# Interface

- Interface única para todo o Dock.
- Os módulos não geram HTML.
- Design consistente.
- Poucos cliques.
- Atualização apenas quando necessária.

---

# Performance

A performance é um requisito obrigatório.

- CPU em idle o mais próximo possível de 0%.
- Evitar polling sempre que exista alternativa baseada em eventos.
- Minimizar consumo de RAM.
- Minimizar acessos ao disco.
- Minimizar acessos à rede.
- Preferir a biblioteca standard do Go.
- Cada dependência deve ser justificada.

Antes de adicionar uma funcionalidade perguntar:

- Melhora realmente o Dock?
- Qual o impacto em CPU?
- Qual o impacto em RAM?
- Existe uma solução mais simples?

---

# Objetivo

Construir uma ferramenta que qualquer administrador de sistemas goste de utilizar diariamente.

