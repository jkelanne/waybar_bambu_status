package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const socketPath = "/tmp/waybar-bambu-status.sock"

var clientsMu sync.Mutex

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
				clientsMu.Lock()
				clients[conn] = struct{}{}
				clientsMu.Unlock()
				go func(c net.Conn) {
					defer c.Close()

					buf := make([]byte, 1)
					for {
						if _, err := c.Read(buf); err != nil {
							clientsMu.Lock()
							delete(clients, c)
							clientsMu.Unlock()
							return
						}
					}
				}(conn)
			}
		}
	}()

	cfgPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln("[ERROR] Unable to get the config directory")
	}

	cfg, err := LoadConfig(fmt.Sprint(cfgPath, "/waybar-bambu-status/config.json"))
	if err != nil {
		log.Fatalf("[ERROR] Something went wrong with LoadConfig: %v", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	opts := mqtt.NewClientOptions().AddBroker(cfg.Printer.Address)
	opts.SetClientID(cfg.Printer.ClientId)
	opts.SetUsername(cfg.Printer.UserName)
	opts.SetPassword(cfg.Printer.AccessCode)
	opts.SetTLSConfig(tlsConfig)

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("[ERROR] MQTT connect failed:", token.Error())
	}

	client.Subscribe(cfg.Printer.MQTTTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		handleMQTTMessage(msg, clients)
	})

	select {}
}

func handleMQTTMessage(msg mqtt.Message, clients map[net.Conn]struct{}) {
	// log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
	var root map[string]any
	if err := json.Unmarshal(msg.Payload(), &root); err != nil {
		log.Println("[ERROR] JSON parse error: ", err)
		return
	}

	printObj, ok := root["print"].(map[string]any)
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
	bedTargetTemp, _ := printObj["bed_target_temper"].(float64)
	chamberTemp, _ := printObj["chamber_temper"].(float64)
	nozzleTargetTemp, _ := printObj["nozzle_target_temper"].(float64)
	nozzleTemp, _ := printObj["nozzle_temper"].(float64)
	layerNum, _ := printObj["layer_num"].(float64)
	totalLayerNum, _ := printObj["total_layer_num"].(float64)

	printPercentage, ok := printObj["mc_percent"].(float64)
	if !ok {
		log.Printf("[%s]: %s", msg.Topic(), msg.Payload())
		log.Printf("Missing or invalid 'mc_percent'")
		return
	}

	remainingTime, ok := printObj["mc_remaining_time"].(float64)
	if !ok {
		log.Printf("Missing or invalid 'mv_remaining_time'")
	}

	subtaskName, ok := printObj["subtask_name"].(string)
	if !ok {
		log.Printf("Missing or invalid 'subtask_name'")
	}

	// 3D Print Icons: 󰹛, 󱇀, 󱢸, 󰹜, 󱇁, 󱢹
	// Apparently there are only 2 gcode_state values RUNNING, FINISH

	var printIcon string
	var output map[string]any
	switch gcodeState {
	case "RUNNING":
		printIcon = "󰹛"
		remTime, err := ConvertTime(remainingTime)
		if err != nil {
			log.Println("Something went wrong with convertTime()")
		}
		output = map[string]any{
			"text": fmt.Sprintf("%s %0.0f%% (%s)", printIcon, printPercentage, remTime),
			"tooltip": fmt.Sprintf("Job: %s\n%s Bed: %0.2f/%0.2f°C\n%s Nozzle: %0.2f/%0.2f°C\n%s Chamber: %0.2f°C\n  Layer: %0.0f/%0.0f",
				subtaskName,
				TemperatureIcon(bedTemp, bedTargetTemp),
				bedTemp,
				bedTargetTemp,
				TemperatureIcon(nozzleTemp, nozzleTargetTemp),
				nozzleTemp,
				nozzleTargetTemp,
				TemperatureIcon(24.0, 100.0),
				chamberTemp,
				layerNum,
				totalLayerNum),
			"class": "running",
		}
	case "FINISH":
		printIcon = "󰹜"
		output = map[string]any{
			"text": fmt.Sprintf("%s IDLE", printIcon),
			"tooltip": fmt.Sprintf("Job: %s\n%s Bed: %0.2f/%0.2f°C\n%s Nozzle: %0.2f/%0.2f°C\n%s Chamber: %0.2f°C\n  Layer: %0.0f/%0.0f",
				subtaskName,
				TemperatureIcon(bedTemp, bedTargetTemp),
				bedTemp,
				bedTargetTemp,
				TemperatureIcon(nozzleTemp, nozzleTargetTemp),
				nozzleTemp,
				nozzleTargetTemp,
				TemperatureIcon(24.0, 100.0),
				chamberTemp,
				layerNum,
				totalLayerNum),
			"class": "idle",
		}
	default:
		printIcon = "󱇁"
		output = map[string]any{
			"text":    fmt.Sprintf("%s %0.0f%%", printIcon, printPercentage),
			"tooltip": fmt.Sprintf("Job: %s\nTemp: %0.2f°C", subtaskName, bedTemp),
			"class":   "fault",
		}
	}

	jsonOutput, err := json.Marshal(output)
	if err == nil {
		fmt.Println(string(jsonOutput))
		clientsMu.Lock()
		for conn := range clients {
			_, err := conn.Write(append(jsonOutput, '\n'))
			if err != nil {
				conn.Close()
				delete(clients, conn)
			}
		}
		clientsMu.Unlock()
	}

}
