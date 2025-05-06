package main

import (
	"crypto/tls"
	// "crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error while loading .env file")
	}
	tlsConfig := &tls.Config{
		// RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}

	opts := mqtt.NewClientOptions().AddBroker(os.Getenv("BL_SSL_ADDR"))
	opts.SetClientID("waybar-printer-status")
	opts.SetUsername("bblp")
	opts.SetPassword(os.Getenv("BL_ACCESS_CODE"))
	opts.SetTLSConfig(tlsConfig)

	opts.OnConnect = func(c mqtt.Client) {
		// if token := c.Subscribe("device/status", 0, onMessage); token.Wait() && token.Error() != nil {
		if token := c.Subscribe("#", 0, onMessage); token.Wait() && token.Error() != nil {
			log.Println("Subscibe error: ", token.Error())
			os.Exit(1)
		}
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("Connect error:", token.Error())
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	client.Disconnect(250)
}

type PrinterStatus struct {
	State   string `json:"state"`
	JobName string `json:"job"`
	Temp    int    `json:"temp"`
}

func onMessage(client mqtt.Client, msg mqtt.Message) {
	// log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
	var root map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &root); err != nil {
		log.Println("JSON parse error: ", err)
		return
	}

	printObj, ok := root["print"].(map[string]interface{})
	if !ok {
		log.Fatal("Missing or invalid 'print' object")
		return
	}

	gcodeState, ok := printObj["gcode_state"].(string)
	if !ok {
		log.Fatal("Missing or invalid 'gcode_state'")
	}

	bedTemp, ok := printObj["bed_temper"].(float64)
	if !ok {
		log.Fatal("Missing or invalid 'bed_temper'")
	}

	printPercentage, ok := printObj["mc_percent"].(float64)
	if !ok {
		log.Fatal("Missing or invalid 'mc_percent'")
	}
	// var status PrinterStatus

	// 3D Print Icons: 󰹛, 󱇀, 󱢸, 󰹜, 󱇁, 󱢹
	// Apparently there are 5 gcode_state values RUNNING, FINISH, PREPARE, PAUSE, or FAILED
	var printIcon string
	switch gcodeState {
	case "RUNNING":
		printIcon = "󰹛"
	case "FINISH":
		printIcon = "󰹜"
	default:
		printIcon = "󱇁"
	}

	output := map[string]interface{}{
		"text":    fmt.Sprintf("%s %0.0f%% (%0.2f°C)", printIcon, printPercentage, bedTemp),
		"tooltip": fmt.Sprintf("Job: %s\nTemp: %0.2f°C", "tmp", bedTemp),
		// "class":   "",
	}

	jsonOutput, err := json.Marshal(output)
	if err == nil {
		fmt.Println(string(jsonOutput))
	}
}
