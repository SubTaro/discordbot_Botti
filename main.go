package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type UserState struct {
	Name    string
	VCState string
}

type ServerChannel struct {
	ServerName string
	SetChannel string
	Run        string
}

var (
	Token       = "YOUR BOT TOKEN"
	BotName     = "YOUR BOT NAME"
	Command     = "!botti"
	Usermap     = map[string]*UserState{}
	Place       string
	helpMessage = `導入ありがとう!
	!botti set: 通知するチャンネルの指定.
	!botti sleep HOGE: HOGE分間通知を行わない. HOGEを入力しなかったら10分間通知を行わない.
	!botti awake: sleep状態の時のbottiを起こして再び通知させる.
	!botti status: 現在のステータスの表示.`

	Servermap = map[string]*ServerChannel{}
)

//TODO: ServerChannel構造体をjsonで保存してbottiを休止しても情報を保持できるようにする

func main() {
	discord, err := discordgo.New(Token)
	if err != nil {
		fmt.Println("Error loging in")
		log.Println(err)
	}

	discord.AddHandler(onVoiceState)
	discord.AddHandler(onCreateMessage)

	err = discord.Open()
	if err != nil {
		fmt.Println("Error loging")
		fmt.Println(err)
	}

	defer discord.Close()

	fmt.Print("Listening ...")

	stopBot := make(chan os.Signal, 1)
	signal.Notify(stopBot, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	<-stopBot

	return
}

//some commands is here
func onCreateMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Printf("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)
	switch {
	//setting command
	//!botti set
	case m.Content == fmt.Sprintf("%s %s", Command, "set"):
		server_guild, err := s.Guild(m.GuildID)
		if err != nil {
			log.Fatal(err)
		}
		_, ok := Servermap[server_guild.Name]
		//if use setting command firstly
		if !ok {
			Servermap[server_guild.Name] = new(ServerChannel)
			Servermap[server_guild.Name].SetChannel = m.ChannelID
			Servermap[server_guild.Name].Run = "Run"
			log.Println("new server added: " + server_guild.Name)
		} else {
			renewChannel(server_guild.Name, m.ChannelID)
		}

		SendMessage(s, Servermap[server_guild.Name].SetChannel, "サーバーのセット完了")

	//!botti help
	//help command
	case m.Content == fmt.Sprintf("%s %s", Command, "help"):
		SendMessage(s, m.ChannelID, helpMessage)

	//!botti sleep
	//sleep command
	case strings.HasPrefix(m.Content, fmt.Sprintf("%s %s", Command, "sleep")):
		arr := strings.Split(m.Content, " ")
		//argument check.
		if len(arr) > 3 {
			SendMessage(s, m.ChannelID, "引数が多すぎます")
		} else {
			server_guild, err := s.Guild(m.GuildID)
			if err != nil {
				log.Fatal(err)
			}
			if len(arr) == 2 {
				go sleepBotti(server_guild.Name, 10)
				SendMessage(s, m.ChannelID, "ZZZ...")
			} else {
				//argument check. Is the 3rd argument number?
				t, err := strconv.Atoi(arr[2])
				if err != nil {
					SendMessage(s, m.ChannelID, "3つめの引数は半角整数を入力してください")
				} else {
					go sleepBotti(server_guild.Name, t)
					SendMessage(s, m.ChannelID, "ZZZ...")
				}
			}
		}
	//!botti status
	//this command can check status
	case m.Content == fmt.Sprintf("%s %s", Command, "status"):
		//make status message

		//get a guild name
		guild, err := s.Guild(m.GuildID)
		if err != nil {
			log.Fatal(err)
		}

		//get a channel
		channel := "UNDEF"
		status := "UNDEF"
		_, ok := Servermap[guild.Name]
		if ok {
			channel = Servermap[guild.Name].SetChannel
			status = Servermap[guild.Name].Run
		}

		message := "Guild: " + guild.Name + "\nChannel: " + channel + "\nStatus: " + status

		//send message
		SendMessage(s, m.ChannelID, message)
	//!botti awake
	//this is the command for waking up botti
	//TODO: sleep中に起こしたときにTickerを止めるためにchannelで起こしたときの処理をsleepbottiに書き込む
	case m.Content == fmt.Sprintf("%s %s", Command, "awake"):
		guild_server, err := s.Guild(m.GuildID)
		if err != nil {
			log.Fatal(err)
		}

		//check to see if it is registering in the Servermap
		_, ok := Servermap[guild_server.Name]
		if !ok {
			SendMessage(s, m.ChannelID, "チャンネルの設定をしてください")
		} else {
			//chage the status
			if Servermap[guild_server.Name].Run == "Run" {
				SendMessage(s, m.ChannelID, "もう起きてるよ！")
			} else {
				Servermap[guild_server.Name].Run = "Run"
				SendMessage(s, m.ChannelID, "おはようなのだ!")
			}
		}
	}
}

func onVoiceState(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	_, ok := Usermap[vs.UserID]
	server_guild, err := s.Guild(vs.GuildID)
	if !ok {
		Usermap[vs.UserID] = new(UserState)
		user, _ := s.User(vs.UserID)
		Usermap[vs.UserID].Name = user.Username
		log.Println("new user added: " + user.Username)
	}
	if err != nil {
		log.Fatal(err)
	}

	if len(vs.ChannelID) > 0 && Usermap[vs.UserID].VCState != vs.ChannelID {
		channel, _ := s.Channel(vs.ChannelID)
		//make a message
		message := Usermap[vs.UserID].Name + "が" + channel.Name + "にきたぞ"
		log.Println(message)

		//if the map has the server of information
		_, exist := Servermap[server_guild.Name]
		if !exist {
			fmt.Println("plz set channelID")
		} else {
			//if the server status is "Run", the bot send message.
			if Servermap[server_guild.Name].Run == "Run" {
				SendMessage(s, Servermap[server_guild.Name].SetChannel, message)
			} else {
				log.Println("server sleep: " + server_guild.Name)
			}
		}
	}

	Usermap[vs.UserID].VCState = vs.ChannelID

	fmt.Printf("%+v\n", vs.VoiceState)
}

func SendMessage(s *discordgo.Session, channelID string, msg string) {
	s.ChannelMessageSend(channelID, msg)
}

func renewChannel(server string, channel string) {
	Servermap[server].SetChannel = channel
}

//this command is fot temporarily stopping this bot
func sleepBotti(guild string, wait_time int) {
	_, ok := Servermap[guild]

	if !ok {
		return
	} else {
		Servermap[guild].Run = "sleep"
		t := time.NewTicker(time.Duration(wait_time*60) * time.Second)

		//wait
		<-t.C
		Servermap[guild].Run = "Run"

		t.Stop()
	}
}
