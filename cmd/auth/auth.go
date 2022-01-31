/*
	Copyright (c) 2022, Network Next, Inc. All rights reserved.

	This is open source software licensed under the BSD 3-Clause License.

	Redistribution and use in source and binary forms, with or without
	modification, are permitted provided that the following conditions are met:

	1. Redistributions of source code must retain the above copyright notice, this
	   list of conditions and the following disclaimer.

	2. Redistributions in binary form must reproduce the above copyright notice,
	   this list of conditions and the following disclaimer in the documentation
	   and/or other materials provided with the distribution.

	3. Neither the name of the copyright holder nor the names of its
	   contributors may be used to endorse or promote products derived from
	   this software without specific prior written permission.

	THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
	AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
	IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
	DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
	FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
	DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
	SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
	CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
	OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
	OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"net"

	"github.com/networknext/udpx/modules/core"
	"github.com/networknext/udpx/modules/envvar"

	"github.com/gorilla/mux"
)

// Allows us to return an exit code and allows log flushes and deferred functions
// to finish before exiting.
func main() {
	os.Exit(mainReturnWithCode())
}

var GatewayAddress *net.UDPAddr
var GatewayPublicKey [core.PublicKeyBytes_Box]byte
var GatewayPrivateKey [core.PrivateKeyBytes_Box]byte
var AuthPrivateKey [core.PrivateKeyBytes_Box]byte

func mainReturnWithCode() int {

	serviceName := "udpx auth"

	core.Info("%s", serviceName)

	// configure

	gatewayAddress, err := envvar.GetAddress("GATEWAY_ADDRESS", core.ParseAddress("127.0.0.1:40000"))
	if err != nil {
		core.Error("invalid GATEWAY_ADDRESS: %v", err)
		return 1
	}

	gatewayPublicKey, err := envvar.GetBase64("GATEWAY_PUBLIC_KEY", nil)
	if err != nil || len(gatewayPublicKey) != core.PublicKeyBytes_Box {
		core.Error("missing or invalid GATEWAY_PUBLIC_KEY: %v", err)
		return 1
	}

	gatewayPrivateKey, err := envvar.GetBase64("GATEWAY_PRIVATE_KEY", nil)
	if err != nil || len(gatewayPrivateKey) != core.PrivateKeyBytes_Box {
		core.Error("missing or invalid GATEWAY_PRIVATE_KEY: %v", err)
		return 1
	}

	authPrivateKey, err := envvar.GetBase64("AUTH_PRIVATE_KEY", nil)
	if err != nil || len(authPrivateKey) != core.PrivateKeyBytes_Box {
		core.Error("missing or invalid AUTH_PRIVATE_KEY: %v", err)
		return 1
	}

	GatewayAddress = gatewayAddress
	copy(GatewayPublicKey[:], gatewayPublicKey[:])
	copy(GatewayPrivateKey[:], gatewayPrivateKey[:])
	copy(AuthPrivateKey[:], authPrivateKey[:])

	// start web server
	{
		router := mux.NewRouter()
		router.HandleFunc("/health", healthHandler).Methods("GET")
		router.HandleFunc("/status", statusHandler).Methods("GET")
		router.HandleFunc("/connect_token", connectTokenHandler).Methods("GET")
		router.HandleFunc("/session_token", sessionTokenHandler).Methods("GET")

		httpPort := envvar.Get("HTTP_PORT", "60000")

		srv := &http.Server{
			Addr:    ":" + httpPort,
			Handler: router,
		}

		go func() {
			core.Info("started http server on port %s", httpPort)
			err := srv.ListenAndServe()
			if err != nil {
				core.Error("failed to start http server: %v", err)
				return
			}
		}()
	}

	// wait for shutdown
	
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, os.Interrupt, syscall.SIGTERM)
	<-termChan

	fmt.Println("shutdown completed")

	return 0
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	_, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello world\n")
}

func connectTokenHandler(w http.ResponseWriter, r *http.Request) {
	var userId [core.UserIdBytes]byte // todo: read in the user id from octet-stream from POST?
	connectToken := core.GenerateConnectToken(userId[:], GatewayAddress, GatewayPublicKey[:], AuthPrivateKey[:], GatewayPublicKey[:])
	w.Header().Set("Content-Type", "application/octet-stream") 
	w.WriteHeader(http.StatusOK)
	w.Write(connectToken)
}

func sessionTokenHandler(w http.ResponseWriter, r *http.Request) {
	
	// todo: read in the current session token from octet-stream from POST
	// https://stackoverflow.com/questions/37462349/sending-octet-stream

	sessionToken := make([]byte, 256)

	w.Header().Set("Content-Type", "application/octet-stream") 
	w.WriteHeader(http.StatusOK)
	w.Write(sessionToken)
}