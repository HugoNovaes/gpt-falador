package main

/* ==============================================================================
Aviso legal: Este software é fornecido "como está", sem garantia de qualquer tipo,
expressa ou implícita, incluindo, mas não se limitando a garantias de adequação a
uma finalidade específica e não violação. Em nenhum caso o autor será responsável
por quaisquer danos diretos, indiretos, incidentais, especiais, exemplares ou
consequenciais (incluindo, mas não se limitando a, aquisição de bens ou serviços
substitutos; perda de uso, dados ou lucros; ou interrupção de negócios)
decorrentes de qualquer forma do uso deste software, mesmo que avisado da
possibilidade de tais danos.

Licença: Este software é distribuído sob a Licença Pública Geral GNU v3.0. Você
pode usar, modificar e/ou redistribuir este software sob os termos da GPL v3.0.
Para mais informações, consulte o arquivo LICENSE.md incluído neste repositório.

Contribuições financeiras são bem-vindas e podem ser feitas através da chave
PIX: 2dc5381e-78d6-4a62-9469-4f50d0ed8a01.

Obrigado!
Hugo S. Novaes
hnovaes@yahoo.com
==============================================================================*/

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

type (
	// Estrutura da mensagem de requisição
	Message struct {
		Role    string `json:"role"`    // Pode ser "user" ou "assistant".
		Content string `json:"content"` // Conteúdo da mensagem.

		// Obs: Como se trata de um chat, o conteúdo do campo Role alterna-se entre "user" e "assistant"
		//      ou seja, o usuário (user) envia e o assistente (assistant) responde.
		//      Para que a IA mantenha o contexto da conversa, deve-se guardar a conversa
		//      em um histórico e enviar à API, sempre que fizer nova pergunta.
	}

	// Estrutura a ser enviada para a API do ChatGPT contendo as mensagens trocadas entre o usuário e a API
	ChatGPTRequest struct {
		Model       string    `json:"model"`       // Atualmente (2023) o model usado é o "gpt-3.5-turbo"
		Messages    []Message `json:"messages"`    // Histórico das mensagens trocadas + nova mensagem.
		Temperature float32   `json:"temperature"` // Valor na faixa de 0.0 a 2.0.
		// Quanto maior o valor de Temperature, mais aleatória é a resposta.
		// Quanto menor, mais determinística.
	}

	// Estrutura retornada pela API do ChatGPT (se responder com sucesso).
	// Mais detalhes no link https://platform.openai.com/docs/api-reference/chat/create
	ChatGPTResult struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Usage   struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
			Index        int    `json:"index"`
		}
	}

	// Estrutura as configurações lidas do arquivo settings.json
	Settings struct {
		URL_API     string  // URL --> "https://api.openai.com/v1/chat/completions"
		API_KEY     string  // Criar API-KEY pelo site https://platform.openai.com/account/api-keys
		GPT_MODEL   string  // Versão atual: gpt-3.5-turbo
		TIMEOUT     int     // Tempo máximo a aguardar por resposta.
		TEMPERATURE float32 // Campo temperature
		TTS         bool    // Se true, fala o texto retornado pela API. Se false, não fala.
		IDIOMA      string  // Idioma do Falador (narrador do texto)

		// Delay máximo para imprimir as palavras na tela. Dependendo do idioma,
		// a pronúncia pode ser mais rápida ou mais lenta. Quando narra números, demora
		// mais para narrá-los do que para imprimir na tela.
		// A narração do idioma em português é mais lenta que em português, por isso,
		// deve-se ajustar o valor desse parâmetro para tentar sincronizar a impressão
		// da resposta com a narração dela.
		MAX_DELAY int
	}
)

const (
	VERSAO = "2023.03.29_gpt-3.5-turbo"

	TECLA_ESC    = 0x1B
	TECLA_ESPACO = 0x20

	// Nome do arquivo de configurações:
	SETTINGS = "settings.json"
)

var (
	mutex            = &sync.Mutex{}      // Mutex para controlar a impressão da frase "ChatGPT está pensando..."
	terminouDePensar = false              // O acesso a essa variável é protegido por esta mutex
	pressionouESC    = false              // Usado para interromper o audio caso esteja narrando o texto retornado.
	noSleep          = false              // Se true, imprime o texto sem pausas.
	printJson        = false              // Se true, imprime o payload retornado pela API.
	interativo       = false              // Se true, executa o GPT no modo interativo (para manter histórico das conversas)
	messages         = make([]Message, 0) // Histórico das mensagens trocadas entre o usuário e a AI
	settings         = &Settings{}        // Armazena as configurações carregadas do arquivo settings.json

	// Para carregar e usar função GetKeyState da API user32.dll do Windows,
	// que verifica o estado de uma tecla qualquer.
	user32_dll  = windows.NewLazyDLL("user32.dll")
	GetKeyState = user32_dll.NewProc("GetKeyState")
)

// Carrega as configurações do arquivo settings.json
func carregaConfiguracoes() {
	s, e := os.ReadFile(SETTINGS)
	if e != nil {
		panic(e)
	}

	e = json.Unmarshal(s, settings)
	if e != nil {
		panic(e)
	}

	messages = messages[:0]
	printSettings()
	fmt.Println("Digite \033[96mhelp\033[m para mais informações")
}

// Grava as configurações no arquivo settings.json
func gravaSettings() {
	bytes, _ := json.MarshalIndent(settings, "", "    ")
	if err := os.WriteFile(SETTINGS, bytes, 0700); err != nil {
		fmt.Println("\033[31m", err.Error(), "\033[m")
	}
}

func printSettings() {
	fmt.Println("GPT Model:\033[96m", settings.GPT_MODEL, "\033[m")
	fmt.Println("Timeout:\033[96m", settings.TIMEOUT, "\033[m")
	fmt.Println("TTS:\033[96m", settings.TTS, "\033[m")
	fmt.Println("Idioma:\033[96m", settings.IDIOMA, "\033[m")
	fmt.Println("Max Delay:\033[96m", settings.MAX_DELAY, "\033[m")
	fmt.Println("Temperature:\033[96m", settings.TEMPERATURE, "\033[m")
}

// Para poder usar o Escape Code para colorir palavras na console, é necessário habilitar primeiro.
func setConsoleColors() error {
	console := windows.Stdout
	var consoleMode uint32
	windows.GetConsoleMode(console, &consoleMode)
	consoleMode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	return windows.SetConsoleMode(console, consoleMode)
}

// Função para alternar o valor da variável terminouDePensar, para interromper
// a goroutine que imprime a frase "ChatGPT está pensando..."
func setTerminouDePensar(b bool) {
	defer mutex.Unlock()
	mutex.Lock()
	terminouDePensar = b
}

// Retorna o valor da variável terminouDePensar. O acesso a essa variável tem que ser
// controloado por mutex, pois, a mesma é usada dentro de uma goroutine.
func getTerminouDePensar() bool {
	defer mutex.Unlock()
	mutex.Lock()
	return terminouDePensar
}

// Imprime o help na tela
func printHelp() {
	fmt.Println("Faça a pergunta para o ChatGPT.")
	fmt.Println("Exemplo: O que pesa mais: um quilo de pena ou um quilo de chumbo?")
	fmt.Println("\r\nTambém pode-se usar os seguintes parâmetros via linha de comando:")
	fmt.Println("\t\033[36m--help\033[m        Exibe estas informações de ajuda.")
	fmt.Println("\t\033[36m--nosleep\033[m     Imprime a resposta de uma só vez, sem delay.")
	fmt.Println("\t              Tecle \033[36mESC\033[m para interromper a impressão da resposta.")
	fmt.Println("\t              Tecle \033[36mESPAÇO\033[m para imprimir a resposta completa sem delay.")
	fmt.Println("\t\033[36m--printjson\033[m   Imprime o conteúdo json retornado pelo servidor (payload)")
	fmt.Println("\t\033[36m--interativo\033[m  Executa este aplicativo no modo interativo, para manter")
	fmt.Println("\t              o histórico da conversa, o que facilita para a IA")
	fmt.Println("\t              contextualizar as próximas perguntas.")
	fmt.Println("\t              Digite \033[36mhelp\033[m para exibir estas informações")
	fmt.Println("\t              Digite \033[36mquit\033[m para terminar o modo interativo")
	fmt.Println("\t              Digite \033[36mreset\033[m para iniciar nova conversa e recarregar")
	fmt.Println("\t              as configurações no arquivo settings.json")
	fmt.Println("\t              Digite \033[36mcls\033[m para limpar a tela (mantém o histórico da conversa)")
	fmt.Println("\t              Digite \033[36mset param=valor\033[m para alterar o valor de algum parâmetro")
	fmt.Println("\t              Exemplo: \033[36mset tts=false\033[m para desativar a fala")
	fmt.Println("\t                       \033[36mset lang=en-us\033[m para alterar o idioma para Inglês dos EUA")
}

// Obtem parâmetros passados via linha de comando ou entra no modo interativo para obter
// as perguntas digitadas pelo usuário na console.
func getPrompt() string {

	if len(os.Args) < 2 {
		interativo = true
	}

	result := ""
	for i := 1; i < len(os.Args); i++ {

		if os.Args[i] == "--help" {
			printHelp()
			continue
		}
		// Verifica se passou o parâmetro --nospeep.
		if os.Args[i] == "--nosleep" {
			noSleep = true
			continue
		}

		// Verifica se passou o parâmetro --printjson.
		if os.Args[i] == "--printjson" {
			printJson = true
			continue
		}

		// Verifica se passou o parâmetro --interativo.
		if os.Args[i] == "--interativo" {
			interativo = true
			continue
		}

		//Caso não seja nenhum dos parâmetros acima, concatena o argumento à variável result.
		result += os.Args[i] + " "
	}

	// Limpa os argumentos para evitar tratamento dos mesmos novamente.
	os.Args = os.Args[:0]
	if interativo && len(result) > 0 {
		return result
	}

	if interativo {
		return getPromptFromConsole()
	}

	return strings.Trim(result, " ")
}

// Modo interativo: aguarda o usuário digitar a frase e teclar enter.
// Também, verifica se o usuário digitou "quit" ou "reset".
// Se digitar "quit", encerra o programa.
// Se digitar "reset", apaga o histórico da conversa - isso faz com que a IA perca o contexto da conversa.
func getPromptFromConsole() string {

	for {
		fmt.Print("\r\n\033[32mPergunta\033[m: ")
		reader := bufio.NewReader(os.Stdin)
		pergunta, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		pergunta = strings.Trim(pergunta, "\r\n\t ")
		comando := strings.ToLower(pergunta)

		switch comando {
		case "cls":
			clearScreen()
			continue
		case "help":
			printHelp()
			return ""
		case "quit":
			os.Exit(0)
		case "reset":
			clearScreen()
			carregaConfiguracoes()
			fmt.Println("Reset efetuado. O histórico e contexto da conversa foi perdido.")
			fmt.Println("Pronto para iniciar outra conversa.")
			continue
		case "set":
			printSettings()
			continue
		}

		// Se o texto digitado contém a palavra "set " (seguido de espaço), aciona o tratamento...
		if strings.HasPrefix(comando, "set ") {
			if trataComandoSet(comando) {
				gravaSettings()
				continue
			}
		}

		if len(pergunta) > 0 {
			return pergunta
		}
	}
}

// Esta função trata os parâmetros do comando "set".
// Se o parâmetro existir e o valor do mesmo for válido, retorna true.
// Caso contrário, retorna false.
func trataComandoSet(comando string) bool {

	// Obtém o parâmetro e o valor após a 4a posição na string "comando", após o "set ".
	tokens := strings.Split(comando[3:], "=")

	// Se não existir um "=" no comando, a quantidade de tokens será menor que 2.
	if len(tokens) < 2 {
		fmt.Printf("\r\n\033[31mComando \"%s\" inválido\033[m\r\n", comando)
		return true
	}

	// Obtém o nome do parâmetro (posição 0) e o valor do parâmetro (posição 1)
	param := strings.Trim(tokens[0], " ")
	valor := strings.Trim(tokens[1], " ")

	// Tratamento para o comando "set model=<modelo>"
	if param == "model" && settings.GPT_MODEL != valor {
		settings.GPT_MODEL = valor
		fmt.Printf("GPT Model alterada para \"%s\"", valor)
		return true
	}

	// Tratamento para o comando "set lang=<idioma>"
	if param == "lang" && settings.IDIOMA != valor {
		settings.IDIOMA = valor
		fmt.Printf("Idioma alterado para \"%s\"", valor)
		return true
	}

	// Tratamento para o comando "set max_delay=<valor>"
	if param == "max_delay" {
		if m, err := strconv.Atoi(valor); err != nil {
			fmt.Printf("\r\n\033[31mValor \"%s\" inválido\033[m\r\n", valor)
		} else if settings.MAX_DELAY != m {
			settings.MAX_DELAY = m
			fmt.Printf("Delay máximo alterado para \"%s\"", valor)
			return true
		}
		return false
	}

	// Tratamento para o comando "set timeout=<valor>"
	if param == "timeout" {
		if m, err := strconv.Atoi(valor); err != nil {
			fmt.Printf("\r\n\033[31mValor \"%s\" inválido\033[m\r\n", valor)
		} else if settings.TIMEOUT != m {
			settings.TIMEOUT = m
			fmt.Printf("Timeout alterado para \"%s\"", valor)
			return true
		}
		return false
	}

	// Tratamento para o comando "set temperature=<valor>"
	if param == "temperature" {
		if m, err := strconv.ParseFloat(valor, 32); err != nil {
			fmt.Printf("\r\n\033[31mValor \"%s\" inválido\033[m\r\n", valor)
		} else if settings.TEMPERATURE != float32(m) {
			settings.TEMPERATURE = float32(m)
			fmt.Printf("Temperature alterado para \"%.2f\"", m)
			return true
		}
		return false
	}

	// Tratamento para o comando "set tts=<valor>"
	if param == "tts" {
		if b, err := strconv.ParseBool(valor); err != nil {
			fmt.Printf("\r\n\033[31mValor \"%s\" inválido\033[m\r\n", valor)
		} else if settings.TTS != b {
			settings.TTS = b
			fmt.Printf("TTS (Text-To-Speech) alterado para \"%s\"", valor)
			return true
		}
		return false
	}
	return false
}

// Limpa a tela quando o usuário digita o comando "cls".
func clearScreen() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// Esta função tem que ser acionada pelo comando "go".
// Aaso contrário, coloca o processo principal em loop infinito.
func pensando() {
	setTerminouDePensar(false)
	timeout := settings.TIMEOUT
	for contador := 0; ; contador++ {

		fmt.Print("\033[93m")
		// A cada múltiplo de 4, limpa a linha e imprime novamente a frase "ChatGPT está pensando"
		if contador%4 == 0 {
			fmt.Printf("\r         \r%d", timeout)
			timeout--
		}

		// Aguarda 1/4 de segundo
		time.Sleep(time.Millisecond * 250)

		// Verifica se terminou de "pensar"
		if timeout == 0 || getTerminouDePensar() {
			break
		}

		// Imprime "." de forma consecutiva para formar, no máximo, os três pontos: "..."
		fmt.Printf("\033[%dm.", 91+rand.Intn(6))
	}
	fmt.Print("\033[m")
}

// Cria uma estrutura de nova requisição e armazena no histórico de mensagens.
func newChatGPTRequest(question string) *ChatGPTRequest {

	// Cria a mensagem com a pergunta do usuário. Instrui a IA a responder no idiomna selecionado.
	msg := Message{
		Role:    "user",
		Content: fmt.Sprintf("%s (You must answer in \"%s\")", question, settings.IDIOMA),
	}

	// Adiciona a mensagem ao histórico de mensagens enviadas/recebidas.
	messages = append(messages, msg)

	return &ChatGPTRequest{
		Model:       settings.GPT_MODEL,
		Messages:    messages,
		Temperature: settings.TEMPERATURE,
	}
}

// Verifica se pressionou e liberou a tecla informada no parâmetro t.
// Chama a função GetKeyState da user32.dll, que verifica o estado da tecla informada.
// Recurso muito útil para varificar se uma tecla foi pressionada sem interromper o loop em que está.
func teclaPressionada(t int32) bool {
	r, _, _ := GetKeyState.Call(uintptr(t))
	return r == 65409 //Código "mágico" que indica que a tecla foi liberada (event KeyUp).
}

// Fala o texto (via audio).
// Se o texto tiver mais que 100 caracteres, o audio é truncado e gera erro.
// Por isso, tem que quebrar em pequenos blocos de no máximo 100 caracteres.
func fala(s string) {

	// Como o ChatGPT responde com marcadores de texto para usar na formatação na tela,
	// alguns desses formatadores são por "acento grave". Esses caracteres são removidos apenas
	// antes de enviar para o narrador. Não afeta na tela (este é formatado antes de imprimir)
	textoSemFormatacao := strings.ReplaceAll(s, "`", "")

	// Transforma o texto em array de palavras para poder somar o tamanho delas e quebrar em
	// blocos de até 100 caracteres em cada bloco, sem quebrar a última palavra.
	palavras := strings.Split(textoSemFormatacao, " ")
	paragrafos := make([]string, 1)
	somaCaracteres := 0

	blocos := 0

	for _, palavra := range palavras {

		// Se o total de aracteres mais o tamanho da palavra atual ultrapassar 100 caracteres,
		// cria novo bloco para as próximas palavras.
		if somaCaracteres+len(palavra) >= 100 {
			paragrafos = append(paragrafos, "")
			blocos++
			somaCaracteres = 0
		}
		paragrafos[blocos] += palavra + " "
		somaCaracteres += len(palavra)
	}

	// Aciona o download dos audios...
	audios := downloadAudios(paragrafos)
	// ... e executa os audios.
	playAudios(audios)
}

// Executa os audios na sequência que foram criados, para manter fluidez e não ser perceptível a troca de
// audios - executa-os sem interrupção.
func playAudios(audios []*DownloadedAudio) error {
	if len(audios) == 0 {
		return errors.New("nenhum audio a reproduzir")
	}

	for _, audio := range audios {

		// Cria o objeto e adiciona os detalhes de execução do mesmo.
		player := Player{
			AudioToPlay:     audio,
			DeleteAfterPlay: true,
		}

		// Executa o audio passando função anônima que irá testar se o mesmo foi interrompido.
		// A interrupção ocorre quando o usuário pressiona ESC durante a impressão da resposta na tela.
		player.Play(func() bool { return pressionouESC })
	}

	return nil
}

// Cria goroutines para baixar os audios para cada bloco de texto de forma concorrente.
func downloadAudios(textos []string) []*DownloadedAudio {
	wg := &sync.WaitGroup{}

	result := make([]*DownloadedAudio, 0)
	for i, s := range textos {
		wg.Add(1)

		// Cria estrutura com os dados de cada arquivo de audio que será baixado para a pasta ./audio
		downloadedAudio := &DownloadedAudio{
			Sequencia: i,
			Path:      fmt.Sprintf("./audio/%d.mp3", i),
			Texto:     s,
		}

		result = append(result, downloadedAudio)
		// Dispara a goroutine de download.
		go downloadFromGoogle(wg, downloadedAudio)
	}
	// Aguarda todas as goroutines terminarem de baixar os audios.
	wg.Wait()
	return result
}

// Envia o bloco de texto para o Google Translate para converter em audio
// Lembrando que o limite de tamanho do texto é de 100 caracteres.
// O parâmetro da QueryString "q" é o texto a ser narrado.
// O parâmetro "tl" (To Language) é o idioma em que o audio será gerado.
func downloadFromGoogle(wg *sync.WaitGroup, downloadedAudio *DownloadedAudio) {
	defer wg.Done()

	// Cria a pasta de destino dos audios baixados.
	dir, err := os.Open("./audio")
	if os.IsNotExist(err) {
		os.MkdirAll("./audio", 0700)
	}

	dir.Close()

	// Transforma o texto em padrão de URL
	txt := url.QueryEscape(downloadedAudio.Texto)
	url := fmt.Sprintf("http://translate.google.com/translate_tts?ie=UTF-8&client=tw-ob&q=%s&tl=%s", txt, settings.IDIOMA)

	// Estabelece a conexão com o site.
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("\033[31m", err.Error(), "\033[m")
		return
	}
	defer response.Body.Close()

	// Cria o arquivo de destino do audio baixado.
	output, err := os.Create(downloadedAudio.Path)
	if err != nil {
		fmt.Println("\033[31m", err.Error(), "\033[m")
		return
	}

	// Copia o conteúdo baixado para o arquivo de destino.
	_, err = io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("\033[31m", err.Error(), "\033[m")
	}
}

// Imprime a resposta na tela.
// Se o parâmetro "--nosleep" for passado, não dá pausas (imprime o texto completo de uma só vez)
func imprimeResposta(s string) {

	// Se o parâmetro TTS (Text-To-Speech) estiver ativo, narra o texto
	if settings.TTS {
		go fala(s)
	}

	// Inicia a variável "acelera" com o valor do parâmetro "--nospeep".
	// Se for "true", imprime os caracteres de forma "lenta", simulando streaming dos mesmos.
	acelera := noSleep

	// Variável imprimiuBlocoCodigo alterna entre true/false quando encontra o marcador "```"
	imprimiuBlocoCodigo := false

	// Variável imprimiuAcentoGrave alterna entre true/false quando encontra o marcador "`"
	imprimiuAcentoGrave := false

	// pressionouESC terá o valor alterado para true se o usuário pressionar ESC durante a impressão da resposta.
	pressionouESC = false

	// countAcentoGrave conterá a quantidade de "`" seguidos. Se for 3, é um marcador de código fonte.
	// Se for 1, é apenas uma referência a um item de código fonte.
	countAcentoGrave := 0

	for _, char := range s {
		if char == '`' {
			countAcentoGrave++
			continue
		}

		if countAcentoGrave == 3 {
			// Alterna a cor para amarelo (cor 33), se já iniciou o bloco de código fonte.
			// ou volta ao normal, se não iniciou.
			if !imprimiuBlocoCodigo {
				fmt.Print("\033[33m")
			} else {
				fmt.Print("\033[m")
			}

			// Alterna entre true e false
			imprimiuBlocoCodigo = !imprimiuBlocoCodigo
			countAcentoGrave = 0
			// Imprime o último caractere antes de retornar para o loop for.
			fmt.Printf("%c", char)
			continue
		}

		if countAcentoGrave == 1 {

			// Alterna a cor para ciano (cor 96), se já iniciou a impressão de trecho entre "`"
			// ou volta ao normal, se não iniciou.
			if !imprimiuAcentoGrave {
				fmt.Print("\033[96m")
			} else {
				fmt.Print("\033[m")
			}
			imprimiuAcentoGrave = !imprimiuAcentoGrave
			countAcentoGrave = 0
			// Imprime o último caractere antes de retornar para o loop for.
			fmt.Printf("%c", char)
			continue
		}

		// Zera o contador de acentos-graves, para não entrar em nenhum dos if's acima.
		countAcentoGrave = 0
		// Cada caractere da string é um rune. Tem que usar %c para converter para caractere.
		fmt.Printf("%c", char)

		if !acelera {
			// Gera uma pausa alearória entre 0 e MAX_DELAY milisegundos entre
			// a impressão da cada caractere para simular streaming das respostas,
			// apesar de ter um parâmetro na estrutura de requisição para tal.
			// Mas, preferi simular.
			tempoPausa := rand.Intn(settings.MAX_DELAY)
			time.Sleep(time.Millisecond * time.Duration(tempoPausa))
		}

		// Verifica se pressionou a tecla ESC, para interromper a impressão do texto
		if teclaPressionada(TECLA_ESC) {
			fmt.Print("\r\n\033[31m <interrompido>\033[m")
			pressionouESC = true
			break
		}

		// Verifica se pressionou a tecla Barra de Espaço, para desativar o delay e imprimir o restante do texto.
		if teclaPressionada(TECLA_ESPACO) {
			acelera = true
		}
	}
}

// Prepara a requisição para enviar à API.
// Retorna um ponteiro para um objeto do tipo http.Request.
func sendRequest(pergunta, tipo string) (*http.Request, context.CancelFunc) {

	chatGPTRequest := newChatGPTRequest(pergunta)

	// Converte a estrutura do objeto para array de bytes no formato json
	reqBytes, _ := json.Marshal(chatGPTRequest)
	reqBody := strings.NewReader(string(reqBytes))

	// Envia a requisição com o método POST para a API do ChatGPT.
	req, _ := http.NewRequest(http.MethodPost, settings.URL_API, reqBody)

	// O resultado tem que ser do tipo application/json
	req.Header.Add("Content-Type", "application/json")

	// Neste ponto que devemos usar a nossa API_KEY
	req.Header.Add("Authorization", "Bearer "+settings.API_KEY)

	// Retorna contexto e ponteiro para função de cancelamento.
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	return req, cancel
}

// Evia a requisição para a API e aguarda a resposta.
func obtemResposta(req *http.Request, c chan *ChatGPTResult, cf context.CancelFunc) {

	// Cria goroutine para gerenciar o tempo de resposta da requisição com o timeout informado no
	// parâmetro TIMEOUT do arquivo settings.json.
	// Se o timeout zerar, aciona a função de cancelamento (a mesma retornada na requisição).
	go func() {
		timeout := settings.TIMEOUT
		for {
			if timeout <= 0 {
				// Chama a função de cancelamento.
				cf()
				return
			}
			// Aguarda 1 segundo para decrementar o timeout
			time.Sleep(time.Second)
			timeout--
		}
	}()

	// Executa a resuisição e aguarda o retorno da mesma.
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		// Testa se o erro foi o acionamento da função de cancelamento (por timeout).
		// Se cancelou por timeout, o texto do erro é "context canceled"
		if strings.Contains(err.Error(), "context canceled") {
			fmt.Printf("\r\033[31mServidor demorou a responder. Envie a pergunta novamente.\033[m")

			// Remove a mensagem do histórico para não repetir a mesma, caso o usuário a reenvie.
			messages = messages[:len(messages)-1]

		} else {
			// Caso o erro não seja o de cancelamento, imprime na tela o erro retornado pela API.
			fmt.Println("\r\n\033[31m", err.Error(), "\033[m")
		}
		// Envia "nil" para o canal, indicando que houve erro.
		c <- nil
		return
	}
	defer res.Body.Close()

	// Lê todo o conteúdo retornado pela API
	retBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("\r\n\033[31m", err.Error(), "\033[m")
		c <- nil
		return
	}

	retorno := &ChatGPTResult{}
	err = json.Unmarshal(retBody, retorno)
	if err != nil {
		// Se ocorrer erro ao converter o json para a estrutura de retorno,
		// pode ser porque retornou uma estrutura de erro, como abaixo.
		fmt.Println("\033[31m", err.Error(), "\033[m")
		c <- nil
		return
	}

	// Se o parâmetro "--printjason" for informado, imprime o retorno na tela.
	if printJson {
		fmt.Print("\r\nJSON retornado: ")
		fmt.Println(string(retBody))
	}

	// Se retornou algum erro na estrutura normal de retorno, o campo Choices vem vazio.
	// E o campo "error" contém os detalhes.
	if len(retorno.Choices) == 0 {
		var erro struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Param   string `json:"param"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		err = json.Unmarshal(retBody, &erro)
		if err != nil {
			fmt.Printf("\r\033[31m%s\033[m\r\n", string(retBody))
		} else {
			fmt.Printf("\r\033[31m%s\033[m\r\n", erro.Error.Message)
		}
		c <- nil
		return
	}

	// Armazena o retorno no histórico de mensagens, para manter o contexto da conversa.
	messages = append(messages, Message{
		Role:    retorno.Choices[0].Message.Role,
		Content: retorno.Choices[0].Message.Content,
	})

	// Envia o conteúdo retornado para o canal
	c <- retorno
}

// A função init() é executada antes da função main().
// Neste momento, carrega o conteúdo do arquivo settings.json e já dá a dica para
// a IA que o idioma das respostas deve ser o informado no campo IDIOMA.
func init() {
	if err := setConsoleColors(); err != nil {
		fmt.Println("Terminal não permite habilitar cores")
	}

	clearScreen()
	fmt.Println("\033[92mGPT-Falador\033[m versão\033[96m", VERSAO, "\033[m")
	fmt.Println("Desenvolvido por Hugo S. Novaes (\033[96mhnovaes@yahoo.com\033[m)")
	fmt.Println("---------------------------------------------------")
	carregaConfiguracoes()
}

func main() {
	for {
		pergunta := getPrompt()

		if len(pergunta) == 0 {
			interativo = true
			continue
		}

		req, cancel := sendRequest(pergunta, "user")
		go pensando()

		c := make(chan *ChatGPTResult)
		go obtemResposta(req, c, cancel)
		retorno := <-c
		setTerminouDePensar(true)
		if retorno == nil {
			continue
		}
		fmt.Print("\r\033[94m        \rResposta\033[m: ")

		for _, b := range retorno.Choices {
			imprimeResposta(b.Message.Content)
		}

		fmt.Println()

		if !interativo {
			break
		}
	}
}
