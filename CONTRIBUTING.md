# Dock - Contributing Guide

## Objetivo

O Dock é um painel de controlo leve, moderno e seguro para executar ações e scripts em servidores, especialmente Raspberry Pi, através de uma interface Web.

O projeto deve ser simples de instalar, fácil de manter e preparado para ser Open Source.

---

## Princípios

- Código simples e legível.
- Segurança em primeiro lugar.
- Desempenho acima de funcionalidades desnecessárias.
- Um único binário.
- Sem dependências pesadas.
- Compatível com Raspberry Pi e Linux.
- Arquitetura preparada para crescer.

---

## Regras de desenvolvimento

### Um passo de cada vez

Nunca executar vários passos importantes em simultâneo.

Depois de cada passo:
- verificar o resultado;
- corrigir eventuais erros;
- só depois avançar.

---

### Criação de ficheiros

Não utilizar editores interativos durante o desenvolvimento.

Utilizar sempre:

- tee
- cat <<'EOF'

para que todos os passos sejam reproduzíveis.

---

### Estrutura

Toda a estrutura do projeto deve permanecer organizada.

Separar claramente:

- servidor HTTP
- configuração
- autenticação
- execução de scripts
- WebSocket
- frontend
- utilitários

---

### Qualidade do código

Antes de adicionar novas funcionalidades:

- manter o código limpo;
- evitar duplicação;
- comentar apenas quando necessário;
- privilegiar nomes claros em vez de comentários.

---

### Dependências

Adicionar apenas dependências realmente necessárias.

Sempre que possível utilizar a biblioteca standard do Go.

---

### Compatibilidade

O Dock deve funcionar em:

- Raspberry Pi (ARM64)
- Linux AMD64

Sempre que possível manter compatibilidade entre arquiteturas.

---

### Docker

O projeto deverá disponibilizar:

- Dockerfile oficial
- imagem multi-arquitetura
- configuração simples

---

### Interface

Objetivos da interface:

- rápida;
- moderna;
- responsiva;
- atualização em tempo real;
- poucos cliques.

Inspirada em ferramentas como Home Assistant, mantendo identidade própria.

---

### Filosofia

Cada funcionalidade deve responder à pergunta:

"Melhora realmente a experiência do utilizador?"

Se a resposta for não, não deve entrar no projeto.


---

## Desempenho e Eficiência

O desempenho não é um objetivo secundário: é um requisito fundamental do Dock.

O Dock foi concebido para funcionar de forma eficiente em equipamentos de baixo consumo, como Raspberry Pi, mantendo uma utilização mínima de CPU e memória.

### Regras obrigatórias

- CPU em idle deve permanecer o mais próximo possível de 0%.
- Evitar polling contínuo; privilegiar arquiteturas orientadas a eventos.
- Não criar goroutines, timers ou tarefas em segundo plano sem necessidade justificada.
- Carregar apenas os componentes estritamente necessários.
- Libertar recursos assim que deixem de ser utilizados.
- Minimizar operações de disco.
- Minimizar acessos à rede.
- Reduzir o número de dependências externas.
- Preferir a biblioteca standard do Go sempre que possível.
- Cada nova dependência deve ter uma justificação técnica.

### Interface

A interface deve ser rápida e responsiva, mesmo em hardware modesto.

- Evitar frameworks pesadas.
- Minimizar JavaScript.
- Reduzir pedidos HTTP.
- Atualizar informação em tempo real apenas quando necessário.

### Filosofia

Antes de implementar qualquer funcionalidade, deve responder-se às seguintes perguntas:

- Melhora realmente a experiência do utilizador?
- Qual o impacto em CPU?
- Qual o impacto em memória?
- Qual o impacto no armazenamento?
- Existe uma solução mais simples?

Se existir uma solução significativamente mais leve, essa deverá ser a opção escolhida.

O Dock pretende distinguir-se por ser uma ferramenta extremamente leve, rápida, segura e estável, adequada para utilização permanente em Raspberry Pi, mini-PCs e servidores Linux.

