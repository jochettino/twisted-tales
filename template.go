package main

import (
	_ "embed"
	"html/template"
	"io"
)

//go:embed templates/card.html
var cardHTML string

var pageTmpl = template.Must(template.New("card").Parse(cardHTML))

type labels struct {
	Card, Of, Back, Next, Show, Hide, Solution, SwitchTo string
	Help, HelpIntro                                      string
	HelpSteps                                            []string
	About, AboutCode, AboutRepo                          string
	AboutPoints                                          []string
}

// repoURL is linked from the About section.
const repoURL = "https://github.com/jochettino/twisted-tales"

var ui = map[string]labels{
	"EN": {
		Card: "Card", Of: "of", Back: "Back", Next: "Next",
		Show: "Show solution", Hide: "Hide solution", Solution: "Solution:", SwitchTo: "Español",
		Help:      "How to play",
		HelpIntro: "Twisted Tales is a game of lateral thinking. Each card is a strange, usually macabre situation, and your group has to work out what really happened.",
		HelpSteps: []string{
			"Pick one player as the Game Master (the Narrator). Only the Game Master looks at the screen.",
			"The Game Master reads the prompt on the card out loud, then privately reads the solution using \"Show solution\".",
			"Everyone else asks questions to piece together the story.",
			"The Game Master may only answer \"Yes\", \"No\" or \"Irrelevant\" (when the question is a dead end).",
			"Keep asking and discussing theories until someone reconstructs the key events. Then the Game Master reads the full solution aloud.",
			"Use \"Next\" and \"Back\" to move between cards. You always start from the prompt.",
		},
		About: "About",
		AboutPoints: []string{
			"This is a personal project.",
			"This app is an experiment, made for testing purposes only. It is not a production-ready app.",
			"Share it with friends.",
		},
		AboutCode: "If you want to see the source code, you can visit my",
		AboutRepo: "GitHub repo",
	},
	"ES": {
		Card: "Carta", Of: "de", Back: "Atrás", Next: "Siguiente",
		Show: "Mostrar solución", Hide: "Ocultar solución", Solution: "Solución:", SwitchTo: "English",
		Help:      "Cómo jugar",
		HelpIntro: "Twisted Tales es un juego de pensamiento lateral. Cada carta es una situación extraña, normalmente macabra, y el grupo tiene que descubrir qué ocurrió realmente.",
		HelpSteps: []string{
			"Elegid a un jugador como Director de Juego (el Narrador). Solo el Director de Juego mira la pantalla.",
			"El Director de Juego lee en voz alta el enunciado de la carta y, en privado, lee la solución con «Mostrar solución».",
			"El resto de jugadores hace preguntas para reconstruir la historia.",
			"El Director de Juego solo puede responder «Sí», «No» o «Irrelevante» (cuando la pregunta no lleva a ninguna parte).",
			"Seguid preguntando y debatiendo teorías hasta que alguien reconstruya los hechos clave. Entonces el Director de Juego lee la solución completa en voz alta.",
			"Usad «Siguiente» y «Atrás» para moveros entre cartas. Siempre empezáis por el enunciado.",
		},
		About: "Acerca de",
		AboutPoints: []string{
			"Este es un proyecto personal.",
			"Esta aplicación es un experimento, creada solo con fines de prueba. No es una aplicación lista para producción.",
			"Compártelo con tus amigos.",
		},
		AboutCode: "Si quieres ver el código fuente, puedes visitar mi",
		AboutRepo: "repositorio de GitHub",
	},
}

// page is the data handed to templates/card.html.
type page struct {
	Card     Card
	Number   int
	Total    int
	Solution bool
	Lang     string
	Other    string // the language the switch button leads to
	L        labels
	Prev     int // 0 when there is no previous card
	Next     int // 0 when there is no next card
	RepoURL  string
}

func renderPage(w io.Writer, p page) error {
	return pageTmpl.Execute(w, p)
}
