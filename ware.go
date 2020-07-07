package main

import (
	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	"github.com/tidusant/c3m-common/mystring"
	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"

	//	"c3m/common/inflect"
	//	"c3m/log"
	"encoding/base64"
	"encoding/json"
	"flag"
	"net"
	"net/rpc"
	"strconv"
	"strings"
)

const (
	defaultcampaigncode string = "XVsdAZGVmYd"
)

type Arith int

func (t *Arith) Run(data string, result *string) error {
	log.Debugf("Call RPCprod args:" + data)
	*result = ""
	//parse args
	args := strings.Split(data, "|")

	if len(args) < 3 {
		return nil
	}
	var usex models.UserSession
	usex.Session = args[0]
	usex.Action = args[2]
	info := strings.Split(args[1], "[+]")
	usex.UserID = info[0]
	ShopID := info[1]
	usex.Params = ""
	if len(args) > 3 {
		usex.Params = args[3]
	}

	//check shop permission
	shop := rpch.GetShopById(usex.UserID, ShopID)
	if shop.Status == 0 {
		*result = c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
		return nil
	}
	usex.Shop = shop

	if usex.Action == "lm" {
		*result = LoadProduct(usex, true)

	} else if usex.Action == "sp" {
		*result = SaveProperty(usex, false)
	} else { //default
		*result = c3mcommon.ReturnJsonMessage("-5", "Action not found.", "", "")
	}

	return nil
}
func SaveProperty(usex models.UserSession, isMain bool) string {
	log.Debugf("param:%s", usex.Params)
	args := strings.Split(usex.Params, ",")
	if len(args) < 2 {
		return c3mcommon.ReturnJsonMessage("-5", "invalid params", "", "")
	}
	var props []models.ProductProperty
	strbytes, _ := base64.StdEncoding.DecodeString(args[1])
	err := json.Unmarshal(strbytes, &props)
	if !c3mcommon.CheckError("create cat parse json", err) {
		return c3mcommon.ReturnJsonMessage("0", "properties parse json fail", "", "")
	}
	//get all product
	prods := rpch.GetAllProds(usex.UserID, usex.Shop.ID.Hex())
	propcodes := make(map[string]string)

	for _, item := range prods {
		for _, prop := range item.Properties {
			propcodes[prop.Code] = prop.Code
		}
	}

	//check new prop code
	for k, prop := range props {
		if strings.Trim(prop.Code, " ") == "" {
			//create new prop code
			for {
				prop.Code = mystring.RandString(4)
				if _, ok := propcodes[prop.Code]; !ok {
					propcodes[prop.Code] = prop.Code
					props[k].Code = prop.Code
					break
				}
			}
		}
	}

	if rpch.SaveProperties(usex.Shop.ID.Hex(), args[0], props) {
		propbytes, _ := json.Marshal(props)
		log.Debugf("json string:%s", string(propbytes))
		return c3mcommon.ReturnJsonMessage("1", "", "Done", string(propbytes))
	}

	return c3mcommon.ReturnJsonMessage("-5", "save properties fail", "", "")

}
func LoadProduct(usex models.UserSession, isMain bool) string {

	prods := rpch.GetAllProds(usex.UserID, usex.Shop.ID.Hex())
	if len(prods) == 0 {
		return c3mcommon.ReturnJsonMessage("2", "", "no prod found", "")
	}

	strrt := "["

	for _, prod := range prods {
		strlang := "{"
		for lang, langinfo := range prod.Langs {
			langinfo.Description = ""
			langinfo.Content = ""
			info, _ := json.Marshal(langinfo)
			strlang += "\"" + lang + "\":" + string(info) + ","
		}
		strlang = strlang[:len(strlang)-1] + "}"
		info, _ := json.Marshal(prod.Properties)
		props := string(info)
		strrt += "{\"Code\":\"" + prod.Code + "\",\"CatId\":\"" + prod.CatId + "\",\"Langs\":" + strlang + ",\"Properties\":" + props + "},"
	}
	strrt = strrt[:len(strrt)-1] + "]"
	log.Debugf("loadprod %s", strrt)
	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)

}
func main() {
	var port int
	var debug bool
	flag.IntVar(&port, "port", 9890, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	//logLevel := log.DebugLevel
	if !debug {
		//logLevel = log.InfoLevel

	}

	// log.SetOutputFile(fmt.Sprintf("adminDash-"+strconv.Itoa(port)), logLevel)
	// defer log.CloseOutputFile()
	// log.RedirectStdOut()

	//init db
	arith := new(Arith)
	rpc.Register(arith)
	log.Infof("running with port:" + strconv.Itoa(port))

	//			rpc.HandleHTTP()
	//			l, e := net.Listen("tcp", ":"+strconv.Itoa(port))
	//			if e != nil {
	//				log.Debug("listen error:", e)
	//			}
	//			http.Serve(l, nil)

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	c3mcommon.CheckError("rpc dail:", err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	c3mcommon.CheckError("rpc init listen", err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}
