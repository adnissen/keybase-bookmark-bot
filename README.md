# keybase bookmark bot
* Install `gcc`
* Log into `keybase`, and the keybase filesystem (`/keybase` or `K:\` for now) 
* Download the dependencies
```
go get github.com/adnissen/go-keybase-chat-bot/kbchat
go get github.com/jinzhu/gorm
go get github.com/jinzhu/gorm/dialects/sqlite
```
* Run the go file

# Usage
To add a bookmark, send a message to yourself with a url as the first part of the message, followed by a description. 
Additional tags may be added with `#`, for example:
```
https://keybase.io crypto for everyone! #crypto #dev
```

If a bookmark has already been saved for the same url, it will be overwritten with the new values.

# Commands
* `.s <search terms>` - search

# Screenshots
![bookmarks!](http://i.imgur.com/KjCUqRk.png)

# Future Stuff (?)
* Pinboard integration
