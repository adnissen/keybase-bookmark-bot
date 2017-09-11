package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/adnissen/go-keybase-chat-bot/kbchat"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Bookmark struct {
	gorm.Model
	Url         string
	Description string
	Tags        string
}

func (b Bookmark) String() string {
	s := ">*url*: " + b.Url + "\\n>*description*: " + b.Description + "\\n>*tags*: _" + b.Tags + "_"
	return s
}

var kbc *kbchat.API
var lastMsgHash uint32
var username string

func fail(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(3)
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func sendMsg(s string) {
	tlfName := fmt.Sprintf("%s,%s", username, username)
	s = strings.Replace(s, "\"", "\\\"", -1)
	fmt.Printf("replying on conversation: %s\n%s", tlfName, s)
	if err := kbc.SendMessageByTlfName(tlfName, s); err != nil {
		fail("Error sending message; %s", err.Error())
	}
	lastMsgHash = hash(s)
}

func main() {
	var kbLoc string
	var kbfsLoc string
	var err error

	flag.StringVar(&kbLoc, "keybase", "keybase", "the location of the Keybase app")
	flag.Parse()

	if kbc, err = kbchat.Start(kbLoc); err != nil {
		fail("Error creating API: %s", err.Error())
	}

	username = kbc.Username()

	//get the location of the kbfs mount so we can store our database there
	//only need to do this on windows, since the location is /keybase otherwise
	if runtime.GOOS == "windows" {
		kbfsLoc = fmt.Sprintf("K:\\private\\%s\\test.db", username)
	} else {
		kbfsLoc = fmt.Sprintf("/keybase/private/%s/test.db", username)
	}

	//dbLoc, err := filepath.Abs(kbfsLoc)
	if err != nil {
		fail("failed to generate file path %s", err.Error())
	}
	fmt.Println("starting with path " + kbfsLoc)

	db, err := gorm.Open("sqlite3", kbfsLoc)
	if err != nil {
		fail("failed to connect database %s %s", kbfsLoc, err.Error())
	}
	defer db.Close()

	db.AutoMigrate(&Bookmark{})

	convos, err := kbc.GetConversations(false)
	if err != nil {
		fail("failed to get conversations: %s", err.Error())
	}

	fmt.Println("waiting for messages")

	var selfConvoId string
	for _, convo := range convos {
		if convo.Channel.Name == username {
			selfConvoId = convo.Id
		}
	}
	if selfConvoId == "" {
		fail("failed to get conversation %s", username)
	}
	c := time.Tick(1200 * time.Millisecond) //sleep 1.2 seconds so we don't hit the api rate limit (900 requests in 15 minutes, so this is just to be safe)
	for range c {
		msgs, err := kbc.GetTextMessages(selfConvoId, false)
		if err != nil {
			fmt.Printf("failed to get messages: %s\n", err.Error())
			continue
		}
		msg := msgs[0].Content.Text.Body
		hashedMsg := hash(msg)
		if lastMsgHash == 0 {
			lastMsgHash = hashedMsg
		}
		if hashedMsg == lastMsgHash {
			continue
		}
		fmt.Println("recieved " + msg)
		lastMsgHash = hashedMsg

		//is this a command we know?
		//.s search
		if strings.HasPrefix(msg, ".s") {
			fmt.Println(".s command")
			if strings.Replace(msg, " ", "", -1) == ".s" {
				sendMsg(">Search: `.s <search terms>`")
				continue
			}
			query := strings.SplitN(msg, " ", 2)
			queryString := ""

			queryString = query[1]
			queryString = "%" + queryString + "%"
			var records []Bookmark
			db.Where("url LIKE ? OR tags LIKE ? OR description LIKE ?", queryString, queryString, queryString).Find(&records)
			for _, r := range records {
				sendMsg(r.String())
			}
			continue
		}

		//split it by space, see if we have a url
		parts := strings.Split(msg, " ")
		start := parts[0]
		//first part url?
		u, err := url.ParseRequestURI(start)
		if err != nil {
			//not a url
		} else {
			//first param is a url
			description := ""
			tags := []string{}
			//use the rest as a description
			for _, seg := range parts[1:] {
				if strings.HasPrefix(seg, "#") {
					tags = append(tags, seg)
					continue
				}
				description = description + " " + seg
			}

			bk := &Bookmark{Url: u.String(), Description: description, Tags: strings.Join(tags, ",")}

			//do we have an existing bookmark for this url?
			var existingBk Bookmark
			db.Where(&Bookmark{Url: bk.Url}).First(&existingBk)
			if &existingBk != nil { //overwrite it
				existingBk.Description = bk.Description
				existingBk.Tags = bk.Tags
				db.Save(existingBk)
			} else { //create it
				db.Create(bk)
			}

			sendMsg(bk.String())
		}
	}
}
