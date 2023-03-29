
# Módulo cliente para a API do ChatGPT
### Desenvolvido em GoLang por Hugo S. Novaes hnovaes@yahoo.com
#
## Uso
#

```gpt [opções] [texto]```

Onde `[opções]` e `[texto]` são opcionais, porém, se informados este cliente do GPT terá o comportamento descrito a seguir.

As `opções` pode ser as seguintes:

```
--help       Mostra as as informações de ajuda.

--nosleep    Imprime a resposta de uma só vez, sem delay.
             Tecle ESC para interromper a impressão da resposta.
             Tecle ESPAÇO para imprimir a resposta completa sem delay.

--printjson  Imprime o conteúdo json retornado pelo servidor (payload).

--interativo Força a execução deste aplicativo no modo interativo, para manter
             o histórico da conversa, o que facilita para a IA
             contextualizar as próximas perguntas.

	         Digite help para exibir estas informações
	         Digite quit para terminar o modo interativo
	         Digite reset para iniciar nova conversa (perde o contexto)
	         e recarregar as configurações no arquivo settings.json.
             Digite cls para limpar a tela (mantém o histórico da conversa)
	         Digite set param=valor para alterar o valor de algum parâmetro.
	         Exemplo: set tts=false para desativar a fala
	                  set lang=en-us para alterar o idioma para Inglês dos EUA
                      set model=gpt-3.5-turbo mara mudar o modelo da IA.
                      set max_delay=120 para mudar o tempo de impressão da resposta.

```

## Exemplo:
```
gpt O que pesa mais: um quilo de pena ou um quilo de chumbo?
```

# O comando `set`:
Usado para mudar as seguintes configurações do arquivo settings.json sem precisar dar reset ou reiniciar o aplicativo. Útil para manter o contexto da conversa.

## Variáveis:
### `tts`

* Alterna entre ativo ou inativo o Text-To-Speech. Os possíveis valores podem ser `true` ou `false`. Se for `true` será narrado o texto retornado pela API do ChatGPT usando a voz do Google. Se for `false`, o texto não será narrado.

* Exemplo: `set tts=true`.

### `lang`
* Altera o idioma da voz do Google. Os códigos dos idiomas estão disponíveis no site https://cloud.google.com/text-to-speech/docs/voices?hl=pt-br (os códigos estão na coluna "Código do idioma" na tabela mostrada nesse site).

* Exemplo: `set lang=en-US`

### `model`
* Altera o modelo do Assistente Virtual ou IA (Inteligência Artificial) que irá responder às suas perguntas. A lista dos modelos disponíveis está no site https://platform.openai.com/docs/models. A versão atual recomendada é a `gpt-3.5-turbo` já que a versão GPT-4 ainda não está disponível até o momento (março/2023).

* Exemplo: `set model=gpt-3.5-turbo`
* Observação: se o modelo selecionado não existir, um erro será retornado ao enviar a pergunta para a IA. O modelo é verificado pelo servidor (a API do ChatGPT) e não por este aplicativo.

### `max_delay`
* Altera o tempo de demora da impressão das respostas na tela. Quanto menor, mais rápido vai imprimir. Um delay entre 160 e 200 foi o mais próximo que encontrei para tentar sincronizar a voz com a impressão do texto em Português do Brasil (pt-BR). Em Inglês dos Estados Unidos (en-US), o tempo que mais se aproximou dessa sincronização foi entre 115 e 150.

### `timeout`
* Altera o tempo para espera por uma resposta da API do ChatGPT, em segundos. Caso esse tempo expire, um erro será apresentado na tela, indicando que o servidor não respondeu. A mensagem enviada não é mantida no histórico das mensagens enviadas, o que significa que não entrará no contexto. Por isso, tem que submeter novamente para o servidor.

* Exemplo: `set timeout=60`

### `temperature`
* Altera o grau de aleatoriedade em que a IA vai responder. O valor dessa parâmetro compreende entre 0.0 e 2.0. O valor 0.0 significa que a resposta é a mais precisa possível. Enquanto que o valor 2.0 é o mais aleatório.

* Exemplo: `set temperature=0.8`
---
# O comando `cls`:
* Use esse comando para limpar a tela. O histórico não é perdido.
---

# O comando `quit`:
* Use esse comando para fechar o aplicativo.
---

# O comando `reset`:
* Use este comando para iniciar uma nova conversa e, também, para reduzir o número de tokens enviados para a API do ChatGPT.
O limite de tokens é 4097. Caso ultrapasse esse valor, retornará a seguinte mensagem de erro: `"This model's maximum context length is 4097 tokens. However, your messages resulted in <num> tokens. Please reduce the length of the messages."`



---
# Alguns detalhes sobre o arquivo de configurações settings.json
##

```
{
    "URL_API": "https://api.openai.com/v1/chat/completions",
    "API_KEY": "informe aqui a sua API_KEY",
    "GPT_MODEL": "gpt-3.5-turbo",
    "TIMEOUT": 120,
    "TEMPERATURE": 0.1,
    "TTS": true,
    "IDIOMA": "pt-BR",
    "MAX_DELAY": 175
}
```

O campo **URL_API** só deve ser alterado se a OpenAPI divulgar um outro canal de comunicação (EndPoint) para este cliente se conectar.
###
O campo **API_KEY** é o código que pode ser obtido no site https://platform.openai.com/account/api-keys para poder comunicar-se com a API do ChatGPT. Cadastre-se nesse site e crie uma ApiKey nele. Copie e cole a chave gerada no campo "API_KEY" do arquivo settings.json.
###

Os demais campos são afetados pelo comando `set` já descrito acima.
###
Contribuições financeiras são bem-vindas e podem ser feitas através da chave
`PIX: 2dc5381e-78d6-4a62-9469-4f50d0ed8a01`
###
Contato com o autor: Hugo S. Novaes - hnovaes@yahoo.com
