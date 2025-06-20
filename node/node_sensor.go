package node

import (
	// "fmt"
	"distributed_system/format"
	"distributed_system/models"
	"distributed_system/utils"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// SensorNode represents a temperature sensor in the system
type SensorNode struct {
	BaseNode
	readInterval   time.Duration
	errorRate      float32
	baseTemp       []float32 // [low, high]
	recentReadings []float32
}

// NewSensorNode creates a new sensor node
func NewSensorNode(id string, interval time.Duration, errorRate float32, baseTempLow float32, baseTempHigh float32) *SensorNode {
	return &SensorNode{
		BaseNode:       NewBaseNode(id, "sensor"),
		readInterval:   interval,
		errorRate:      errorRate,
		baseTemp:       []float32{baseTempLow, baseTempHigh},
		recentReadings: make([]float32, 0, 15),
	}
}

func (s *SensorNode) InitVectorClockWithSites(sites []string) {
	s.mu.Lock()
	s.vectorClock = make([]int, len(sites))
	s.nodeIndex = utils.FindIndex(s.ctrlLayer.GetName(), sites)
	s.vectorClockReady = true
	s.mu.Unlock()
}

// Start begins the sensor's operation
func (s *SensorNode) Start() error {
	format.Display(format.Format_d(
		"Start()", "node_sensor.go",
		"Starting sensor node "+s.GetName()))

	go func() {
		s.mu.Lock()
		vcReady := s.vectorClockReady
		s.mu.Unlock()
		for !vcReady {
			time.Sleep(500 * time.Millisecond) // Wait until vector clock is ready
			s.mu.Lock()
			vcReady = s.vectorClockReady
			s.mu.Unlock()
		}
		s.isRunning = true
	}()

	go func() {
		for {
			// continue until the vector clock is ready
			s.mu.Lock()
			vectorClockReady := s.vectorClockReady
			s.mu.Unlock()
			if !vectorClockReady {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			// ✅ Incrémenter l'horloge vectorielle locale
			s.mu.Lock()
			s.clk += 1
			s_clk_str := strconv.Itoa(s.clk)
			s.vectorClock[s.nodeIndex] += 1
			s_VC_str := utils.SerializeVectorClock(s.vectorClock)
			s.mu.Unlock()

			// Générer une lecture
			reading := s.generateReading()

			// ✅ Stocker dans le FIFO local
			s.mu.Lock()
			if len(s.recentReadings) >= 15 {
				s.recentReadings = s.recentReadings[1:]
			}
			s.recentReadings = append(s.recentReadings, float32(reading.Temperature))
			s.mu.Unlock()

			// Créer le message
			msg_id := s.GenerateUniqueMessageID()
			msg := format.Msg_format_multi(format.Build_msg_args(
				"id", msg_id,
				"propagation", "true",
				"type", "new_reading",
				"sender_name", s.GetName(),
				"sender_name_source", s.GetName(),
				"sender_type", s.Type(),
				"destination", "applications",
				"content_type", "sensor_reading",
				"vector_clock", s_VC_str,
				"clk", s_clk_str,
				"item_id", fmt.Sprintf("%s_%s", s.GetName(), s_clk_str),
				"content_value", strconv.FormatFloat(float64(reading.Temperature), 'f', -1, 32),
			))

			s.logFullMessage(msg_id, reading)

			// Envoi vers couche application
			format.Display_w(s.GetName(), "Start()", "Sending message to application layer: "+msg)
			if (*s.ctrlLayer).SendApplicationMsg(msg) == nil {
				s.mu.Lock()
				s.nbMsgSent = s.nbMsgSent + 1
				s.mu.Unlock()
			}

			time.Sleep(s.readInterval)
		}
	}()
	select {} // Block forever
}

// generateReading produces a simulated temperature reading
func (s *SensorNode) generateReading() models.Reading {
	// Generate a base realistic temperature (between low and high values).
	baseTemp := rand.Float32()*(s.baseTemp[1]-s.baseTemp[0]) + s.baseTemp[0]

	// Sometimes introduce errors based on errorRate
	if rand.Float32() < s.errorRate {
		// Generate erroneous readings (very high)
		baseTemp = baseTemp + 50.0 + rand.Float32()*100.0
	}

	// Add some minor natural variation
	temperature := baseTemp + (rand.Float32()-0.5)*2.0

	return models.Reading{
		// ReadingID:   s.GenerateUniqueMessageID(),
		Temperature: temperature,
		Timestamp:   time.Now(),
		SensorID:    s.ID(),
		IsVerified:  false,
	}
}

func (s *SensorNode) initLogFile() {
	filename := "node_" + s.ID() + "_log.txt"
	f, err := os.Create(filename)
	if err == nil {
		defer f.Close()
		f.WriteString("# Log de node " + s.ID() + " créé à " + time.Now().Format(time.RFC3339) + "\n")
	}
}

func (s *SensorNode) logFullMessage(msg_id string, reading models.Reading) {
	filename := "node_" + s.ID() + "_log.txt"

	destination := "applications" // Par défaut

	logLine := "/" + "id=" + msg_id +
		"/type=new_reading" +
		"/sender_name=" + s.GetName() +
		"/sender_name_source=" + s.GetName() +
		"/sender_type=" + s.Type() +
		"/destination=" + destination +
		"/clk=" + strconv.Itoa(s.clk) +
		// "/vector_clock=" + utils.SerializeVectorClock(s.vectorClock) +
		"/content_type=sensor_reading" +
		"/content_value=" + strconv.FormatFloat(float64(reading.Temperature), 'f', -1, 32)
	// "\n"
	if s.vectorClockReady {
		logLine = logLine + "/vector_clock=" + utils.SerializeVectorClock(s.vectorClock)
	}

	logLine = logLine + "\n"

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(logLine)
	}
}

func (s *SensorNode) ID() string   { return s.id }
func (s *SensorNode) Type() string { return s.nodeType }

func (s *SensorNode) HandleMessage(channel chan string) {
	for msg := range channel {
		// 🔎 Identifier le type de message
		vcStr := format.Findval(msg, "vector_clock")
		recvVC, err := utils.DeserializeVectorClock(vcStr)

		if len(recvVC) != len(s.vectorClock) {
			recvVC = append(recvVC, make([]int, len(s.vectorClock)-len(recvVC))...)
		}

		// 🔁 Mettre à jour le vector clock à la réception
		s.mu.Lock()
		if s.vectorClockReady {
			if err == nil {
				for i := 0; i < len(s.vectorClock); i++ {
					s.vectorClock[i] = utils.Max(s.vectorClock[i], recvVC[i])
				}
				if len(s.vectorClock) > s.nodeIndex {
					s.vectorClock[s.nodeIndex] += 1
				}
			}
		}
		recClk, _ := strconv.Atoi(format.Findval(msg, "clk"))
		s.clk = utils.Synchronise(s.clk, recClk)

		s.mu.Unlock()

	}
}

func (v *SensorNode) SendMessage(msg string, toHandleMessageArgs ...bool) {
	toHandleMessage := false
	if len(toHandleMessageArgs) > 0 {
		toHandleMessage = toHandleMessageArgs[0]
	}
	v.mu.Lock()
	v.vectorClock[v.nodeIndex]++
	serializedClock := utils.SerializeVectorClock(v.vectorClock)
	v_clk := v.clk
	v.mu.Unlock()

	if v.vectorClockReady {
		msg = format.Replaceval(msg, "vector_clock", serializedClock)
	}
	msg = format.Replaceval(msg, "clk", strconv.Itoa(v_clk))
	msg = format.Replaceval(msg, "id", v.GenerateUniqueMessageID())

	if toHandleMessage {
		// v.ctrlLayer.HandleMessage(msg)
		v.channel_to_ctrl <- msg
	} else {
		v.ctrlLayer.SendApplicationMsg(msg)
	}

	// Increment the number of messages sent
	// (used in ID generation, for next messages)
	v.mu.Lock()
	v.nbMsgSent++
	v.mu.Unlock()

}

func (n *SensorNode) GetLocalState() string {
	n.mu.Lock()
	readings := make([]string, len(n.recentReadings))
	for i, val := range n.recentReadings {
		readings[i] = strconv.FormatFloat(float64(val), 'f', 2, 32)
	}
	n.mu.Unlock()
	return strings.Join(readings, ", ")
}

func (n *SensorNode) SetSnapshotInProgress(inProgress bool) {
}
