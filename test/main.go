package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/skratchdot/open-golang/open"
)

// Desired Resource = https://management.core.windows.net/

func codeReceiver(port int, finished chan bool) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		log.Printf("body: %v", string(body))
		err := r.ParseForm()
		if err != nil {
			log.Fatal(err)
		}
		params := r.Form
		log.Printf("params: %v", params)

		var buf bytes.Buffer
		body, err = json.Marshal(params)
		if err != nil {
			log.Fatal(err)
		}
		json.HTMLEscape(&buf, body)
		w.Header().Add("Content-Type", "application/json")
		w.Write(buf.Bytes())
		log.Error(string(body))

		finished <- true
	})
	http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
}

func main() {
	port, err := getFreePort()
	if err != nil {
		log.Fatal(err)
	}
	finished := make(chan bool)
	go codeReceiver(port, finished)
	open.Start(fmt.Sprintf("https://login.microsoftonline.com/common/oauth2/authorize?response_type=code&client_id=04b07795-8ddb-461a-bbee-02f9e1bf7b46&redirect_uri=http://localhost:%v&state=n6p0punuhtl5neyl4dez&resource=https://management.core.windows.net/&prompt=select_account", port))
	<-finished
	time.Sleep(100 * time.Millisecond)

	//
	//        # exchange the code for the token
	//        context = self._create_auth_context(tenant)
	//        token_entry = context.acquire_token_with_authorization_code(results['code'], results['reply_url'],
	//                                                                    resource, _CLIENT_ID, None)
	//        self.user_id = token_entry[_TOKEN_ENTRY_USER_ID]
	//        logger.warning("You have logged in. Now let us find all the subscriptions to which you have access...")
	//        if tenant is None:
	//            result = self._find_using_common_tenant(token_entry[_ACCESS_TOKEN], resource)
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
