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
	"bytes"
	"os"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
)

type (
	DownloadedAudio struct {
		Sequencia int
		Path      string
		Texto     string
		Erro      error
		Playing   bool
	}

	Player struct {
		AudioToPlay     *DownloadedAudio
		DeleteAfterPlay bool
		player          oto.Player
	}
)

// Trecho extraido e adaptado do https://github.com/hegedustibor/htgo-tts
// O que tem de diferente?
// 1 - Recebe como parâmetro uma função que é acionada para verificar se é para dar stop no player.
// 2 - Usa a estrutura Player com os detalhes do audio a executar.
// 3 - Se o campo DeleteAfterPlayer da estrutura for true, deleta o arquivo ao terminar de tocar.
func (p *Player) Play(stopFunc func() bool) error {
	defer p.Close()

	fileName := p.AudioToPlay.Path
	fileBytes, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	fileBytesReader := bytes.NewReader(fileBytes)

	decodedMp3, err := mp3.NewDecoder(fileBytesReader)
	if err != nil {
		return err
	}

	numOfChannels := 2
	audioBitDepth := 2

	otoCtx, readyChan, err := oto.NewContext(decodedMp3.SampleRate(), numOfChannels, audioBitDepth)
	if err != nil {
		return err
	}
	<-readyChan

	p.player = otoCtx.NewPlayer(decodedMp3)

	p.player.Play()

	for p.player.IsPlaying() {

		if stopFunc != nil {
			if stopFunc() {
				break
			}
		}

		time.Sleep(time.Millisecond * 10)
	}

	return p.Close()
}

func (p *Player) IsPlaying() bool {
	return p.player != nil && p.player.IsPlaying()
}

func (p *Player) Close() error {
	if p.player == nil {
		return nil
	}

	if p.DeleteAfterPlay && p.AudioToPlay != nil {
		os.Remove(p.AudioToPlay.Path)
	}

	return p.player.Close()
}
