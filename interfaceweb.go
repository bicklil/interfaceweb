package interfaceweb

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gyuho/goraph"
)

const temps = 200

/*DataRecue est une interface a implementer pour faire marcher l'affichage*/
type DataRecue interface {
	FnConvertDataToSend() (int, string)
}

type page struct {
	Node []string
	Link [][]string
}

var contenu page

type paquetWebSocket struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var msg []byte
var lock sync.Mutex

func updateInf(d DataRecue, fin chan bool) {

	var a paquetWebSocket
	a.ID, a.Text = d.FnConvertDataToSend()
	// mettre une file
	lock.Lock()
	msg, _ = json.Marshal(a)
	lock.Unlock()
	fin <- true

}

func boucle(can DataRecue) {
	fin := make(chan bool)
	for {
		time.Sleep(temps * time.Millisecond)
		go updateInf(can, fin)
		<-fin

	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("../interfaceweb/graphTemp.html")
	if err != nil {
		fmt.Println(err)
		panic("chagement")
	}
	err = tmpl.Execute(w, contenu)
	if err != nil {
		fmt.Println(err)
	}
}

/*InitAffichage permet de creer le serveur local et d'initialiser les champs
pour afficher le graph
La fonction sert a mettre en forme les données a envoyer a l'affichage*/
func InitAffichage(g goraph.Graph, tab []string, affichage DataRecue) {
	for n := range tab {
		contenu.Node = append(contenu.Node, "{id: "+strconv.Itoa(n)+", label: '"+tab[n]+","+strconv.Itoa(n)+"'},\n")
	}
	contenu.Link = make([][]string, len(tab))
	for n := range tab {
		suivant, _ := g.GetTargets(goraph.StringID(tab[n]))
		for j := range suivant {
			indVois := -1
			for i := range tab {
				if tab[i] == j.String() {
					indVois = i
				}
			}
			contenu.Link[n] = append(contenu.Link[n], "{from:"+strconv.Itoa(n)+", to: "+strconv.Itoa(indVois)+", arrows:'to'},\n")
		}
	}
	// on enleve ,\n au dernier element
	dElement := contenu.Link[len(contenu.Link)-1][len(contenu.Link[len(contenu.Link)-1])-1]
	contenu.Link[len(contenu.Link)-1][len(contenu.Link[len(contenu.Link)-1])-1] = dElement[:len(dElement)-2]
	fmt.Println(temps)
	go boucle(affichage)

	http.HandleFunc("/", handler)
	http.HandleFunc("/vis.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../interfaceweb/node_modules/vis/dist/vis.js")
	})
	http.HandleFunc("/vis.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../interfaceweb/node_modules/vis/dist/vis.css")
	})

	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			time.Sleep(temps * time.Millisecond)

			// Write message to browser
			lock.Lock()
			if msg != nil {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					fmt.Println("veuillez quitter le programme, navigateur fermé")
					return
				}
				msg = nil
			}
			lock.Unlock()
		}
	})
	exec.Command("xdg-open", "http://localhost:8000").Start()
	log.Fatal(http.ListenAndServe(":8000", nil))

}
