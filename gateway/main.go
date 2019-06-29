package gateway

//Run function is the main Entry
func Run(configPath string) {
	c := (&config{}).load(configPath)
	us = (&udpService{
		config: *c,
	}).init()
	ss = (&syncService{
		config: *c,
	}).init()
	ws = (&webServer{
		config: *c,
	}).init()

	if c.IsServer {
		its = (&itsManager{
			config: *c,
		}).init()
	}

	us.addService("syncService", ss)
	us.addService("webService", ws)

	us.run()
}
