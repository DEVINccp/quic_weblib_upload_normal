package regist


import (
	"encoding/json"
	"fmt"
	"net/http"
)

func Regist(defaultZone, appName string, port, renewalInterval, durationInterval int) {
	// create eureka client

	client := NewClient(&Config{
		DefaultZone:           defaultZone,
		App:                   appName,
		Port:                  port,
		RenewalIntervalInSecs: renewalInterval,
		DurationInSecs:        durationInterval,
		Metadata: map[string]interface{}{
			"VERSION":              "0.1.0",
			"NODE_GROUP_ID":        0,
			"PRODUCT_CODE":         "DEFAULT",
			"PRODUCT_VERSION_CODE": "DEFAULT",
			"PRODUCT_ENV_CODE":     "DEFAULT",
			"SERVICE_VERSION_CODE": "DEFAULT",
		},
	})
	// start client, register、heartbeat、refresh
	client.Start()

	// http server
	http.HandleFunc("/v1/services", func(writer http.ResponseWriter, request *http.Request) {
		// full applications from eureka server
		apps := client.Applications

		b, _ := json.Marshal(apps)
		_, _ = writer.Write(b)
	})

	// start http server
	if err := http.ListenAndServe(":10000", nil); err != nil {
		fmt.Println(err)
	}
}
