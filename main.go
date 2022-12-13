package main

import (
	"fmt"
	"log"
	"encoding/json"
	"io/ioutil"
	"os"
	"bytes"
)

var Core = struct {
	*Keypair
	*Blockchain
	*Network
}{}

func getKeypair(fileName string) *Keypair {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		// File or directory does not exist
		fmt.Println("Generating keypair...")
		keypair := GenerateNewKeypair()

		buffer := new(bytes.Buffer)
		encoder := json.NewEncoder(buffer)
		encoder.SetIndent("", "\t")

		err := encoder.Encode(keypair)
		if err != nil {
			fmt.Println("save fail.")
			return keypair
		}
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			fmt.Println("save fail.")
			return keypair
		}
		_, err = file.Write(buffer.Bytes())
		if err != nil {
			fmt.Println("save fail.")
			return keypair
		}

		return keypair
	} else {
		// Some other error such as missing permissions
		fmt.Println("Read keypair from ", fileName, " ...")
		keypair := new(Keypair)
		file, _ := ioutil.ReadFile(fileName)
		json.Unmarshal(file, keypair)
		return keypair
	}
}

func Start(address string) {

	// Setup keys

	// keypair, _ := OpenConfiguration(HOME_DIRECTORY_CONFIG)
	// if keypair == nil {

	// 	fmt.Println("Generating keypair...")
	// 	keypair = GenerateNewKeypair()
	// 	WriteConfiguration(HOME_DIRECTORY_CONFIG, keypair)
	// }

	keypair := getKeypair("./local_keypair.json")
	Core.Keypair = keypair

	// Setup Network
	Core.Network = SetupNetwork(address, BLOCKCHAIN_PORT)
	go Core.Network.Run()
	for _, n := range SEED_NODES() {
		Core.Network.ConnectionsQueue <- n
	}

	// Setup blockchain
	Core.Blockchain = SetupBlockchan()
	go Core.Blockchain.Run()

	go func() {
		for {
			select {
			case msg := <-Core.Network.IncomingMessages:
				HandleIncomingMessage(msg)
			}
		}
	}()
}

func main() {
	Start("localhost:3333")
}

func CreateTransaction(txt string) *Transaction {

	t := NewTransaction(Core.Keypair.Public, nil, []byte(txt))
	t.Header.Nonce = t.GenerateNonce(TRANSACTION_POW)
	t.Signature = t.Sign(Core.Keypair)

	return t
}

func HandleIncomingMessage(msg Message) {

	switch msg.Identifier {
	case MESSAGE_SEND_TRANSACTION:
		t := new(Transaction)
		_, err := t.UnmarshalBinary(msg.Data)
		if err != nil {
			networkError(err)
			break
		}
		Core.Blockchain.TransactionsQueue <- t

	case MESSAGE_SEND_BLOCK:
		b := new(Block)
		err := b.UnmarshalBinary(msg.Data)
		if err != nil {
			networkError(err)
			break
		}
		Core.Blockchain.BlocksQueue <- *b
	}
}

func logOnError(err error) {

	if err != nil {
		log.Println("[Todos] Err:", err)
	}
}
