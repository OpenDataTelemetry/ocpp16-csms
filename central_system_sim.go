package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"

	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/firmware"
	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/localauth"
	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/reservation"
	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/lorenzodonini/ocpp-go/ws"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

const (
	defaultListenPort          = 8887
	defaultHeartbeatInterval   = 600
	envVarServerPort           = "SERVER_LISTEN_PORT"
	envVarTls                  = "TLS_ENABLED"
	envVarCaCertificate        = "CA_CERTIFICATE_PATH"
	envVarServerCertificate    = "SERVER_CERTIFICATE_PATH"
	envVarServerCertificateKey = "SERVER_CERTIFICATE_KEY_PATH"
)

var MQTT_BROKER = os.Getenv("MQTT_BROKER")

var log *logrus.Logger
var centralSystem ocpp16.CentralSystem

type EvseDown struct {
	Name   string `json:"name"`
	Tags   Tags `json:"tags"`

}
type Tags struct {
	ConnectorId   string `json:"connectorId"`
	// FeatureName   string `json:"featureName"`
	// DeviceId      string `json:"deviceId"`
	// EvseId        string `json:"evseId"`
	// DeviceType    string `json:"deviceType"`
	// ChargePointId string `json:"chargePointId"`
	// Timestamp     int64  `json:"timestamp"`
}

func setupCentralSystem() ocpp16.CentralSystem {
	return ocpp16.NewCentralSystem(nil, nil)
}

func setupTlsCentralSystem() ocpp16.CentralSystem {
	var certPool *x509.CertPool
	// Load CA certificates
	caCertificate, ok := os.LookupEnv(envVarCaCertificate)
	if !ok {
		log.Infof("no %v found, using system CA pool", envVarCaCertificate)
		systemPool, err := x509.SystemCertPool()
		if err != nil {
			log.Fatalf("couldn't get system CA pool: %v", err)
		}
		certPool = systemPool
	} else {
		certPool = x509.NewCertPool()
		data, err := os.ReadFile(caCertificate)
		if err != nil {
			log.Fatalf("couldn't read CA certificate from %v: %v", caCertificate, err)
		}
		ok = certPool.AppendCertsFromPEM(data)
		if !ok {
			log.Fatalf("couldn't read CA certificate from %v", caCertificate)
		}
	}
	certificate, ok := os.LookupEnv(envVarServerCertificate)
	if !ok {
		log.Fatalf("no required %v found", envVarServerCertificate)
	}
	key, ok := os.LookupEnv(envVarServerCertificateKey)
	if !ok {
		log.Fatalf("no required %v found", envVarServerCertificateKey)
	}
	server := ws.NewTLSServer(certificate, key, &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  certPool,
	})
	return ocpp16.NewCentralSystem(nil, server)
}

// Run for every connected Charge Point, to simulate some functionality
// func exampleRoutine(chargePointID string, handler *CentralSystemHandler) {
// 	// Wait for some time
// 	time.Sleep(2 * time.Second)
// 	// Reserve a connector
// 	reservationID := 42
// 	clientIdTag := "l33t"
// 	connectorID := 1
// 	expiryDate := types.NewDateTime(time.Now().Add(1 * time.Hour))
// 	cb1 := func(confirmation *reservation.ReserveNowConfirmation, err error) {
// 		if err != nil {
// 			logDefault(chargePointID, reservation.ReserveNowFeatureName).Errorf("error on request: %v", err)
// 		} else if confirmation.Status == reservation.ReservationStatusAccepted {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("connector %v reserved for client %v until %v (reservation ID %d)", connectorID, clientIdTag, expiryDate.FormatTimestamp(), reservationID)
// 		} else {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't reserve connector %v: %v", connectorID, confirmation.Status)
// 		}
// 	}
// 	e := centralSystem.ReserveNow(chargePointID, cb1, connectorID, expiryDate, clientIdTag, reservationID)
// 	if e != nil {
// 		logDefault(chargePointID, reservation.ReserveNowFeatureName).Errorf("couldn't send message: %v", e)
// 		return
// 	}
// 	// Wait for some time
// 	time.Sleep(1 * time.Second)
// 	// Cancel the reservation
// 	cb2 := func(confirmation *reservation.CancelReservationConfirmation, err error) {
// 		if err != nil {
// 			logDefault(chargePointID, reservation.CancelReservationFeatureName).Errorf("error on request: %v", err)
// 		} else if confirmation.Status == reservation.CancelReservationStatusAccepted {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("reservation %v canceled successfully", reservationID)
// 		} else {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("couldn't cancel reservation %v", reservationID)
// 		}
// 	}
// 	e = centralSystem.CancelReservation(chargePointID, cb2, reservationID)
// 	if e != nil {
// 		logDefault(chargePointID, reservation.ReserveNowFeatureName).Errorf("couldn't send message: %v", e)
// 		return
// 	}
// 	// Wait for some time
// 	time.Sleep(5 * time.Second)
// 	// Get current local list version
// 	cb3 := func(confirmation *localauth.GetLocalListVersionConfirmation, err error) {
// 		if err != nil {
// 			logDefault(chargePointID, localauth.GetLocalListVersionFeatureName).Errorf("error on request: %v", err)
// 		} else {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("current local list version: %v", confirmation.ListVersion)
// 		}
// 	}
// 	e = centralSystem.GetLocalListVersion(chargePointID, cb3)
// 	if e != nil {
// 		logDefault(chargePointID, localauth.GetLocalListVersionFeatureName).Errorf("couldn't send message: %v", e)
// 		return
// 	}
// 	// Wait for some time
// 	time.Sleep(5 * time.Second)
// 	configKey := "MeterValueSampleInterval"
// 	configValue := "10"
// 	// Change meter sampling values time
// 	cb4 := func(confirmation *core.ChangeConfigurationConfirmation, err error) {
// 		if err != nil {
// 			logDefault(chargePointID, core.ChangeConfigurationFeatureName).Errorf("error on request: %v", err)
// 		} else if confirmation.Status == core.ConfigurationStatusNotSupported {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Warnf("couldn't update configuration for unsupported key: %v", configKey)
// 		} else if confirmation.Status == core.ConfigurationStatusRejected {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Warnf("couldn't update configuration for readonly key: %v", configKey)
// 		} else {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("updated configuration for key %v to: %v", configKey, configValue)
// 		}
// 	}
// 	e = centralSystem.ChangeConfiguration(chargePointID, cb4, configKey, configValue)
// 	if e != nil {
// 		logDefault(chargePointID, localauth.GetLocalListVersionFeatureName).Errorf("couldn't send message: %v", e)
// 		return
// 	}

// 	// Wait for some time
// 	time.Sleep(5 * time.Second)
// 	// Trigger a heartbeat message
// 	cb5 := func(confirmation *remotetrigger.TriggerMessageConfirmation, err error) {
// 		if err != nil {
// 			logDefault(chargePointID, remotetrigger.TriggerMessageFeatureName).Errorf("error on request: %v", err)
// 		} else if confirmation.Status == remotetrigger.TriggerMessageStatusAccepted {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("%v triggered successfully", core.HeartbeatFeatureName)
// 		} else if confirmation.Status == remotetrigger.TriggerMessageStatusRejected {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("%v trigger was rejected", core.HeartbeatFeatureName)
// 		}
// 	}
// 	e = centralSystem.TriggerMessage(chargePointID, cb5, core.HeartbeatFeatureName)
// 	if e != nil {
// 		logDefault(chargePointID, remotetrigger.TriggerMessageFeatureName).Errorf("couldn't send message: %v", e)
// 		return
// 	}

// 	// Wait for some time
// 	time.Sleep(5 * time.Second)
// 	// Trigger a diagnostics status notification
// 	cb6 := func(confirmation *remotetrigger.TriggerMessageConfirmation, err error) {
// 		if err != nil {
// 			logDefault(chargePointID, remotetrigger.TriggerMessageFeatureName).Errorf("error on request: %v", err)
// 		} else if confirmation.Status == remotetrigger.TriggerMessageStatusAccepted {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("%v triggered successfully", firmware.GetDiagnosticsFeatureName)
// 		} else if confirmation.Status == remotetrigger.TriggerMessageStatusRejected {
// 			logDefault(chargePointID, confirmation.GetFeatureName()).Infof("%v trigger was rejected", firmware.GetDiagnosticsFeatureName)
// 		}
// 	}
// 	e = centralSystem.TriggerMessage(chargePointID, cb6, firmware.DiagnosticsStatusNotificationFeatureName)
// 	if e != nil {
// 		logDefault(chargePointID, remotetrigger.TriggerMessageFeatureName).Errorf("couldn't send message: %v", e)
// 		return
// 	}

// }


// func for subscribeRoutine
func connLostHandler(c MQTT.Client, err error) {
	fmt.Printf("Connection lost, reason: %v\n", err)
	os.Exit(1)
}

// func for subscribeRoutine
// func defineChargingPointConnectorId(deviceId string) (chargingPointId string, connectorId int) {

// 	if deviceId == "BRIMTE19400577" {
// 		chargingPointId = "BRIMTS01"
// 		connectorId = 1
// 	} else if deviceId == "BRIMTE19743013" {
// 		chargingPointId = "BRIMTS01"
// 		connectorId = 2
// 	}

// 	return chargingPointId, connectorId
// }

func subscribeRoutine(chargePointID string, handler *CentralSystemHandler) {
	id := uuid.New().String()
	// ORGANIZATION := os.Getenv("ORGANIZATION")
	// DEVICE_TYPE := os.Getenv("DEVICE_TYPE")

	// MqttSubscriberClient
	var sbMqttSubClientId strings.Builder
	sbMqttSubClientId.WriteString("parse-evse-sub-")
	sbMqttSubClientId.WriteString(id)

	// MqttSubscriberTopic
	var sbMqttSubTopic strings.Builder
	// sbMqttSubTopic.WriteString("debug/OpenDataTelemetry/")
	// sbMqttSubTopic.WriteString("IMT/EVSE/UnlockConnector/BRIMTE19400577/down/+")
	sbMqttSubTopic.WriteString("IMT/EVSE/+/")
	sbMqttSubTopic.WriteString(chargePointID)
	sbMqttSubTopic.WriteString("/down/+")
	// sbMqttSubTopic.WriteString(DEVICE_TYPE)
	// sbMqttSubTopic.WriteString(DEVICE_TYPE)
  // sbMqttSubTopic.WriteString("+/+/+/+/+/+")
	// sbMqttSubTopic.WriteString("#")
	// sbMqttSubTopic.WriteString("/+/+/+")

	// MQTT
	mqttSubBroker := MQTT_BROKER
	mqttSubClientId := sbMqttSubClientId.String()
	mqttSubUser := "public"
	mqttSubPassword := "public"
	mqttSubQos := 0

	mqttSubOpts := MQTT.NewClientOptions()
	mqttSubOpts.AddBroker(mqttSubBroker)
	mqttSubOpts.SetClientID(mqttSubClientId)
	mqttSubOpts.SetUsername(mqttSubUser)
	mqttSubOpts.SetPassword(mqttSubPassword)
	mqttSubOpts.SetConnectionLostHandler(connLostHandler)

	c := make(chan [2]string)

	mqttSubOpts.SetDefaultPublishHandler(func(mqttClient MQTT.Client, msg MQTT.Message) {
		c <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	mqttSubClient := MQTT.NewClient(mqttSubOpts)
	if token := mqttSubClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		fmt.Printf("Connected to %s\n", mqttSubBroker)
	}

	if token := mqttSubClient.Subscribe(sbMqttSubTopic.String(), byte(mqttSubQos), nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// MQTT
	for {
		// 1. Input
		incoming := <-c

		// 2. Process
		// 2.1. Process Topic
		s := strings.Split(incoming[0], "/")
		// OpenDataTelemetry/IMT/LNS/MEASUREMENT/DEVICE_ID/up/imt
		// OpenDataTelemetry/IMT/LNS/MEASUREMENT/DEVICE_ID/down/chirpstackv4
		organization := s[0]
		deviceType := s[1]
		measurement := s[2]
		// chargingPointId := s[3]
		var evseDown EvseDown
		json.Unmarshal([]byte(incoming[1]), &evseDown)

		connectorID, _ := (strconv.ParseInt(evseDown.Tags.ConnectorId, 10, 64))
		// direction := s[4]
		// etc := s[5]

		// // DEBUG
		// measurement := s[4]
		// deviceId := s[5]
		// direction := s[6]
		// etc := s[7]

		

		switch organization {
		case "IMT":
			switch deviceType {
			case "EVSE":
				// m := parseEvseFeatureName(measurement, deviceId, incoming[1])
				switch measurement {
				case "RemoteStopTransaction":
					fmt.Printf("Entrou RemoteStopTransaction")
					// Remote stop
					// Wait for some time
					// chargePointID, connectorID := defineChargingPointConnectorId(deviceId)

					time.Sleep(2 * time.Second)
					cbStop := func(confirmation *core.RemoteStopTransactionConfirmation, err error) {
						if err != nil {
							// Handle error
						} else if confirmation.Status == "RemoteStopFailed" {
							// Stop failed
						} else {
							// Success
						}
					}
					e := centralSystem.RemoteStopTransaction(chargePointID, cbStop, int(connectorID))
					if e != nil {
						logDefault(chargePointID, reservation.ReserveNowFeatureName).Errorf("couldn't send message: %v", e)
						return
					}

				case "UnlockConnector":

					// Reserve a connector
					// chargePointID, connectorID := defineChargingPointConnectorId(deviceId)

					//unlock connector
					// Wait for some time
					time.Sleep(1 * time.Second)
					// Cancel the reservation
					cbUnlock := func(confirmation *core.UnlockConnectorConfirmation, err error) {
						if err != nil {
							// Handle error
						} else if confirmation.Status == core.UnlockStatusUnlockFailed {
							// Unlock failed
						} else if confirmation.Status == core.UnlockStatusNotSupported {
							// Unlock not supported by charge point
						} else {
							// Success
						}
					}
					e := centralSystem.UnlockConnector(chargePointID, cbUnlock, int(connectorID))
					if e != nil {
						logDefault(chargePointID, core.UnlockConnectorFeatureName).Errorf("couldn't send message: %v", e)
						return
					}
				}

			default:
			}

		}

		fmt.Printf("\n>>>>")
		fmt.Printf("\nTopic: %s", incoming[0])
		fmt.Printf("\n>>>>")
	}
}
// Start function
func main() {
	// TODO: MAKE A CHANNEL WITH 4 FIELDS: [type, chargePointID, connectorId, data]
	// TODO: USE defineDeviceId(connectorId) TO MOUNT THE TOPIC
	// TODO: RECEIVE EACH VARIABLE AND MOUNT THE JSON MESSAGE
	// <- Make channel and assign to var
	c = make(chan string)
	c2 = make(chan [2]string)

	id := uuid.New().String()
	var sbMqttClientId strings.Builder
	sbMqttClientId.WriteString("ocpp-")
	sbMqttClientId.WriteString(id)

	// pBroker := "mqtt://mqtt.maua.br:1883"
	// pBroker := "mqtt://smartcampus.maua.br:1883"
	pBroker := MQTT_BROKER

	pClientId := sbMqttClientId.String()
	pUser := "PUBLIC"
	pPassword := "public"
	pQos := 0

	pOpts := MQTT.NewClientOptions()
	pOpts.AddBroker(pBroker)
	pOpts.SetClientID(pClientId)
	pOpts.SetUsername(pUser)
	pOpts.SetPassword(pPassword)

	pClient := MQTT.NewClient(pOpts)
	if token := pClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		fmt.Printf("Connected to %s\n", pBroker)
	}

	go func() {
		for { // send MQTT
			// incoming, ok := <-c2
			incoming := <-c2
			token := pClient.Publish(incoming[0], byte(pQos), false, incoming[1])
			token.Wait()

		}
	}()

	// Load config from ENV
	var listenPort = defaultListenPort
	port, _ := os.LookupEnv(envVarServerPort)
	if p, err := strconv.Atoi(port); err == nil {
		listenPort = p
	} else {
		log.Printf("no valid %v environment variable found, using default port", envVarServerPort)
	}
	// Check if TLS enabled
	t, _ := os.LookupEnv(envVarTls)
	tlsEnabled, _ := strconv.ParseBool(t)
	// Prepare OCPP 1.6 central system
	if tlsEnabled {
		centralSystem = setupTlsCentralSystem()
	} else {
		centralSystem = setupCentralSystem()
	}
	// Support callbacks for all OCPP 1.6 profiles
	handler := &CentralSystemHandler{chargePoints: map[string]*ChargePointState{}}
	centralSystem.SetCoreHandler(handler)
	centralSystem.SetLocalAuthListHandler(handler)
	centralSystem.SetFirmwareManagementHandler(handler)
	centralSystem.SetReservationHandler(handler)
	centralSystem.SetRemoteTriggerHandler(handler)
	centralSystem.SetSmartChargingHandler(handler)

	// Add handlers for dis/connection of charge points
	centralSystem.SetNewChargePointHandler(func(chargePoint ocpp16.ChargePointConnection) {
		handler.chargePoints[chargePoint.ID()] = &ChargePointState{connectors: map[int]*ConnectorInfo{}, transactions: map[int]*TransactionInfo{}}
		log.WithField("client", chargePoint.ID()).Info("new charge point connected")
		// go exampleRoutine(chargePoint.ID(), handler)
		go subscribeRoutine(chargePoint.ID(), handler)
	})
	centralSystem.SetChargePointDisconnectedHandler(func(chargePoint ocpp16.ChargePointConnection) {
		log.WithField("client", chargePoint.ID()).Info("charge point disconnected")
		delete(handler.chargePoints, chargePoint.ID())
	})
	ocppj.SetLogger(log.WithField("logger", "ocppj"))
	ws.SetLogger(log.WithField("logger", "websocket"))
	// Run central system
	log.Infof("starting central system on port %v", listenPort)
	centralSystem.Start(listenPort, "/{ws}")
	log.Info("stopped central system")
}

func init() {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	// Set this to DebugLevel if you want to retrieve verbose logs from the ocppj and websocket layers
	log.SetLevel(logrus.InfoLevel)
}
