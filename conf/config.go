package conf

const WsPath = "/wsConnection"
const CheckDomain = "check"
const WebsiteHostName = "view"

var ARecordPrefixes = map[string]string{
	"n2.":      "35.217.6.93",
	"*.check.": "35.217.6.93",
}

var NsRecordPrefixes = map[string]string{
	"check.": "n2.",
}

var SoaRecordPrefixes = map[string]string{
	"check.": "300 IN SOA n2.DOMAIN. hostmaster.DOMAIN. 1 21600 3600 259200 300",
}

const DefaultARecord = "*.check."

const Ttl = "30"

var DEBUG = true
