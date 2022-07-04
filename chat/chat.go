package chat

import (
	"fmt"
	"goChat/utils"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var DB *gorm.DB

type dbBank struct {
	Id uint `json:"id"`
	Sender string `json:"sender"`
	Body string `json:"body"`
}

type Chat struct {
	users    map[string]*User
	messages chan *Message
	join     chan *User
	leave    chan *User
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		log.Printf("%s %s%s %v\n", r.Method, r.Host, r.RequestURI, r.Proto)
		return r.Method == http.MethodGet
	},
}

func (c *Chat) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	keys := r.URL.Query()
	username := keys.Get("username")
	if strings.TrimSpace(username) == "" {
		username = fmt.Sprintf("anom-%d", utils.GetRandomI64())
	}

	user := &User{
		Username: username,
		Conn:     conn,
		Global:   c,
	}

	c.join <- user

	user.Read()
}

func (c *Chat) Run() {
	for {
		select {
		case user := <-c.join:
			c.add(user)
		case message := <- c.messages:
			c.broadcast(message)
		case user := <- c.leave:
			c.disconnect(user)
		}
	}
}

func (c *Chat) add(user *User) {
	if _, ok := c.users[user.Username]; !ok {
		c.users[user.Username] = user
		log.Printf("Added user: %s , Total: %d\n", user.Username, len(c.users))
		body := fmt.Sprintf("%s join the chat", user.Username)
		c.broadcast(NewMessage(body, "Server"))

		messageSend := dbBank{
			Sender: "Server",
			Body: body,
		}

		DB.Create(&messageSend)
	}
}

func (c *Chat) broadcast(message *Message) {
	log.Printf("Broadcast message: %v\n", message)
	for _, user := range c.users {
		user.Write(message)
	}
}

func (c *Chat) disconnect(user *User) {
	if _, ok := c.users[user.Username]; ok {
		defer user.Conn.Close()
		delete(c.users, user.Username)
		log.Printf("User left the chat: %s, Total: %d\n", user.Username, len(c.users))
		body := fmt.Sprintf("%s left the chat", user.Username)
		c.broadcast(NewMessage(body, "Server"))

		messageSend := dbBank{
			Sender: "Server",
			Body: body,
		}

		DB.Create(&messageSend)
	}
}


func Start(port string) {

	log.Printf("Chat is listening on http://localhost:%s\n", port)

	c := &Chat{
		users:    make(map[string]*User),
		messages: make(chan *Message),
		join:     make(chan *User),
		leave:    make(chan *User),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to Go Webchat!"))
	})

	http.HandleFunc("/chat", c.Handler)

	connection, err := gorm.Open(mysql.Open("root:root@/golang-webchat-messages"), &gorm.Config{})
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/golang-webchat-messages")

	if err != nil {
		panic("could not connect to the database")
	}

	DB = connection

	go c.Run()

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	app.Get("/get-messages", func(c *fiber.Ctx) error {
		res, err := db.Query("SELECT * FROM db_banks")
		if err != nil {
			log.Println(err)
		}
		defer res.Close()

		messages := []dbBank{}

		for res.Next(){
			p := dbBank{}
			err := res.Scan(&p.Id, &p.Sender, &p.Body)
			if err != nil{
				fmt.Println(err)
				continue
			}
			messages = append(messages, p)
		}

		// // var allMessages dbBank
		// // result := DB.Find(&allMessages)

		return c.JSON(messages)
	})

	go app.Listen(":8000")
	go log.Fatal(http.ListenAndServe(port, nil))
}
