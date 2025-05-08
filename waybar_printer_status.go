package main

import (
	"bufio"
	"crypto/tls"
	// "crypto/tls"
	"net"

	"encoding/json"
	"fmt"
	"log"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const socketPath = "/tmp/waybar-printer.sock"

type Config struct {
	Printer struct {
		Address    string `json:"address"`
		AccessCode string `json:"access_code"`
		MQTTTopic  string `json:"mqtt_topic"`
		Serial     string `json:"serial"`
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %s", path)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

type PrinterStatus struct {
	State   string `json:"state"`
	JobName string `json:"job"`
	Temp    int    `json:"temp"`
}

func main() {
	// Try to connect as a client
	if conn, err := net.Dial("unix", socketPath); err == nil {
		log.Println("Running as a client")
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		return
	}

	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to create socket: %v", err)
	}
	defer os.Remove(socketPath)

	log.Println("Running as MQTT master")

	clients := make(map[net.Conn]struct{})

	go func() {
		for {
			conn, err := listener.Accept()
			if err == nil {
				clients[conn] = struct{}{}
				go func(c net.Conn) {
					defer c.Close()
					buf := make([]byte, 1)
					for {
						if _, err := c.Read(buf); err != nil {
							delete(clients, c)
							return
						}
					}
				}(conn)
			}
		}
	}()

	cfgPath, err := os.UserConfigDir()
	if err != nil {
		log.Printf("Unable to get the config directory")
		return
	}

	cfg, err := LoadConfig(fmt.Sprint(cfgPath, "/waybar-printer/config.json"))
	if err != nil {
		log.Printf("Something went wrong with LoadConfig: %v", err)
		return
	}

	tlsConfig := &tls.Config{
		// RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}

	opts := mqtt.NewClientOptions().AddBroker(cfg.Printer.Address)
	opts.SetClientID("waybar-printer-status")
	opts.SetUsername("bblp")
	opts.SetPassword(cfg.Printer.AccessCode)
	opts.SetTLSConfig(tlsConfig)

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT connect failed:", token.Error())
	}

	client.Subscribe(cfg.Printer.MQTTTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		// log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
		var root map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &root); err != nil {
			log.Println("JSON parse error: ", err)
			return
		}

		printObj, ok := root["print"].(map[string]interface{})
		if !ok {
			log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
			log.Printf("Missing or invalid 'print' object")
			return
		}

		gcodeState, ok := printObj["gcode_state"].(string)
		if !ok {
			log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
			log.Printf("Missing or invalid 'gcode_state'")
			return
		}

		bedTemp, ok := printObj["bed_temper"].(float64)
		if !ok {
			log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
			log.Printf("Missing or invalid 'bed_temper'")
			return
		}

		printPercentage, ok := printObj["mc_percent"].(float64)
		if !ok {
			log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
			log.Printf("Missing or invalid 'mc_percent'")
			return
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
			for conn := range clients {
				conn.Write(jsonOutput)
				conn.Write([]byte("\n"))
			}
		}

	})

	select {}

	// opts.OnConnect = func(c mqtt.Client) {
	// 	// if token := c.Subscribe("device/status", 0, onMessage); token.Wait() && token.Error() != nil {
	// 	if token := c.Subscribe("#", 0, onMessage); token.Wait() && token.Error() != nil {
	// 		log.Println("Subscibe error: ", token.Error())
	// 		os.Exit(1)
	// 	}
	// }
	//
	// client := mqtt.NewClient(opts)
	// if token := client.Connect(); token.Wait() && token.Error() != nil {
	// 	log.Println("Connect error:", token.Error())
	// 	os.Exit(1)
	// }

	// sigs := make(chan os.Signal, 1)
	// signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// <-sigs
	// client.Disconnect(250)
}
