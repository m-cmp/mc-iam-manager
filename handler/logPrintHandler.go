package handler

import "log"

func LogPrintHandler(ad string, info interface{}) {
	log.Println("======== " + ad + " ========")
	log.Println(info)
	log.Println("======== " + ad + " ========")
}
