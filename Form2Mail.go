package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	netUrl "net/url"
	"os"
	"strings"
	"time"
)

var (
	listenAddr       string
	accessKeyId      string
	accessKeySecret  string
	fromEmailAddress string
	fromEmailAlias   string
	emailSubject     string
	toEmailAddress   string
)

type (
	responseWriter struct {
		http.ResponseWriter
		statusCode int
	}
)

func randomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func urlEncode(input string) string {
	return netUrl.QueryEscape(strings.Replace(strings.Replace(strings.Replace(input, "+", "%20", -1), "*", "%2A", -1), "%7E", "~", -1))
}

func sendMail(content string) error {
	v := netUrl.Values{}
	v.Set("Format", "json")
	v.Set("Version", "2015-11-23")
	v.Set("AccessKeyId", accessKeyId)
	v.Set("SignatureMethod", "HMAC-SHA1")
	v.Set("Timestamp", time.Now().UTC().Format(time.RFC3339))
	v.Set("SignatureVersion", "1.0")
	v.Set("SignatureNonce", randomString(64))
	v.Set("Action", "SingleSendMail")
	v.Set("AccountName", fromEmailAddress)
	v.Set("ReplyToAddress", "false")
	v.Set("AddressType", "0")
	v.Set("FromAlias", fromEmailAlias)
	v.Set("Subject", emailSubject)
	v.Set("TextBody", content)
	v.Set("ToAddress", toEmailAddress)

	h := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	h.Write([]byte("POST&%2F&" + urlEncode(v.Encode())))
	v.Set("Signature", base64.StdEncoding.EncodeToString(h.Sum(nil)))

	req, err := http.NewRequest("POST", "https://dm.aliyuncs.com/", strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{
		Timeout: time.Duration(3 * time.Second),
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseMultipartForm(2 << 20)
		var content string
		for k := range r.PostForm {
			content += k + " => " + strings.Join(r.PostForm[k], ", ") + "\n"
		}
		if err := sendMail(content); err != nil {
			errorJson(w, err.Error(), http.StatusBadRequest)
		} else {
			errorJson(w, "OK", http.StatusOK)
		}
		return
	}
	errorJson(w, "Please use POST", http.StatusMethodNotAllowed)
}

func errorJson(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]string{}
	resp["message"] = msg
	json.NewEncoder(w).Encode(&resp)
}

func realIp(r *http.Request) string {
	return r.Header.Get("X-Real-Ip")
}

func log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rW := &responseWriter{w, http.StatusOK}
		handler.ServeHTTP(rW, r)
		queries, _ := json.Marshal(r.PostForm)
		end := time.Now()
		fmt.Fprintf(os.Stderr,
			"%s [%s] [%s] %d %s %s %s\n",
			end.Format(time.RFC3339),
			realIp(r),
			end.Sub(start),
			rW.statusCode,
			r.Method,
			r.URL.String(),
			queries,
		)
	})
}

func init() {
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:8080", "Listen to address")
	flag.StringVar(&accessKeyId, "akid", "", "Access key ID")
	flag.StringVar(&accessKeySecret, "aksecret", "", "Access key secret")
	flag.StringVar(&fromEmailAddress, "from", "", "From email address")
	flag.StringVar(&fromEmailAlias, "alias", "", "From email alias")
	flag.StringVar(&emailSubject, "subject", "", "Email subject")
	flag.StringVar(&toEmailAddress, "to", "", "To email address")
}

func main() {
	flag.Parse()

	http.HandleFunc("/Form2Mail", handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		errorJson(w, "No such route", http.StatusNotFound)
	})
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Fprintf(os.Stderr, "Started listening on %s\n", listenAddr)
		fmt.Fprintln(os.Stderr, http.Serve(listener, log(http.DefaultServeMux)))
	}
	os.Exit(1)
}
