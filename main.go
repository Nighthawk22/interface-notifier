package main

import (
	"io/ioutil"
	"log"
	"net"
	"os"

	"gopkg.in/yaml.v2"
)

//NetInterface struct for interface name and ip
type NetInterface struct {
	Name string
	IP   []string
}

type SlackConfig struct {
	URL      string `yaml:"url"`
	Channel  string `yaml:"channel"`
	Username string `yaml:"username"`
	Icon     string `yaml:"icon"`
}

type EmailConfig struct {
	To           string `yaml:"to" json:"to"`
	From         string `yaml:"from" json:"from"`
	Host         string `yaml:"smarthost,omitempty" json:"smarthost,omitempty"`
	AuthUsername string `yaml:"auth_username" json:"auth_username"`
	AuthPassword string `yaml:"auth_password" json:"auth_password"`
	RequireTLS   bool   `yaml:"require_tls,omitempty" json:"require_tls,omitempty"`
}

type NotificationReceiver struct {
	SlackConfig *SlackConfig `yaml:"slack,omitempty"`
	EmailConfig *EmailConfig `yaml:"email,omitempty"`
}

type Config struct {
	Interfaces []string               `yaml:"interfaces"`
	Receivers  []NotificationReceiver `yaml:"receivers"`
}

func addIPToInterfaces(interfaces *[]NetInterface, ifaceName string, ip string) {
	found := false
	for _, inface := range *interfaces {
		if inface.Name == ifaceName {
			inface.IP = append(inface.IP, ip)
			found = true
			break
		}
	}
	if !found {
		ips := make([]string, 1)
		ips[0] = ip
		*interfaces = append(*interfaces, NetInterface{ifaceName, ips})
	}
}

func getInterfaceIPs() ([]NetInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var interfaces []NetInterface

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			addIPToInterfaces(&interfaces, i.Name, ip.String())
		}
	}
	return interfaces, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func isInterfaceInConfig(name string, list []string) bool {
	for _, item := range list {
		if name == item {
			return true
		}
	}
	return false
}

func difference(slice1 []string, slice2 []string) []string {
	var diff []string

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}

	return diff
}

func sendNotification(newInterface NetInterface, config Config) {
	log.Print("Found change, sending: ", newInterface)
	for _, receiver := range config.Receivers {
		if receiver.SlackConfig != nil {
			log.Print("Send to Slack", newInterface)
		}
		if receiver.EmailConfig != nil {
			log.Print("Send to Email", newInterface)
		}
	}
}

func isNewInterfaceInOldInterfaces(name string, oldinterfaces []NetInterface) bool {
	for _, netinterface := range oldinterfaces {
		if name == netinterface.Name {
			return true
		}
	}
	return false
}

func getOldInterfaceByName(name string, oldinterfaces []NetInterface) *NetInterface {
	for _, netinterface := range oldinterfaces {
		if name == netinterface.Name {
			return &netinterface
		}
	}
	return nil
}

func sendNotificationOnInterfaceChange(newInterfaces []NetInterface, oldInterfaces []NetInterface, config Config) {
	for _, newInterface := range newInterfaces {
		// No old interfaces. Sending all new ips in config
		if len(oldInterfaces) == 0 && isInterfaceInConfig(newInterface.Name, config.Interfaces) {
			sendNotification(newInterface, config)
			continue
		}

		// Interface is not in old interfaces but in config
		if !isNewInterfaceInOldInterfaces(newInterface.Name, oldInterfaces) && isInterfaceInConfig(newInterface.Name, config.Interfaces) {
			sendNotification(newInterface, config)
			continue
		}

		// There was an old interface,  the ip changed and it is in config
		if oldInterface := getOldInterfaceByName(newInterface.Name, oldInterfaces); oldInterface != nil && isInterfaceInConfig(oldInterface.Name, config.Interfaces) && len(difference(oldInterface.IP, newInterface.IP)) > 0 {
			sendNotification(newInterface, config)
		}
	}
}

func loadConfig(fileName string) (*Config, error) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil

}

func main() {
	config, err := loadConfig("./config.yaml")

	if err != nil {
		log.Fatal(err)
	}

	interfaces, err := getInterfaceIPs()
	filePath := os.TempDir() + "/interface-notifier.gob"

	if err != nil {
		log.Fatal("Could not fetch interfaces: ", err)
	}
	var previousInterfaces []NetInterface
	prevExists, err := exists(filePath)

	if err != nil {
		log.Fatal(err)
	}
	if prevExists {
		err = Load(filePath, &previousInterfaces)
		if err != nil {
			log.Fatal(err)
		}
	}
	sendNotificationOnInterfaceChange(interfaces, previousInterfaces, *config)
	err = Save(filePath, &interfaces)
	if err != nil {
		log.Fatal(err)
	}

}
