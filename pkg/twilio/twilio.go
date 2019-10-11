package twilio

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"fuse/pkg/domain"

	"github.com/parnurzeal/gorequest"
	log "github.com/sirupsen/logrus"
)

type TwilioClient struct {
	phoneTo   string // make call to this phone
	phoneFrom string // .. from this phone

	token string // api token
	sid   string // api sid

	twimlUrl string // url which returns TwiML document
}

type TwiML struct {
	XMLName xml.Name `xml:"Response"`

	Say  string `xml:",omitempty"`
	Play string `xml:",omitempty"`
}

func NewTwilioClient(phoneTo, phoneFrom, token, sid, twimlUrl string) *TwilioClient {
	return &TwilioClient{
		phoneTo:   phoneTo,
		phoneFrom: phoneFrom,
		token:     token,
		sid:       sid,
		twimlUrl:  twimlUrl,
	}
}

/*
 * Implements HTTP callback server for slash-command in slack.
 */
func (t *TwilioClient) Start() error {
	twimlUrlParsed, err := url.Parse(t.twimlUrl)
	if err != nil {
		log.WithField("url", t.twimlUrl).Fatal("twilio: can't parse twimlUrl")
	}

	http.HandleFunc(twimlUrlParsed.Path, func(w http.ResponseWriter, r *http.Request) {
		xml, err := t.generateTwiML()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		w.Write(xml)
	})

	return http.ListenAndServe(":7778", nil) // TODO: configurable port?
}

func (t *TwilioClient) generateTwiML() ([]byte, error) {
	// TODO: replace with public available mp3
	twiml := TwiML{Play: "http://ont.pw/tuntrol.mp3"}

	xml, err := xml.Marshal(twiml)
	if err != nil {
		log.WithError(err).Error("twilio: error during TwiML serialization")
		return nil, err
	}

	return []byte(xml), nil
}

func (t *TwilioClient) GetName() string {
	return "twilio"
}

func (t *TwilioClient) Crit(msg domain.Message) error {
	// Build out the data for our message
	request := map[string]interface{}{
		"To":   t.phoneTo,
		"From": t.phoneFrom,
		"Url":  t.twimlUrl,
	}

	res, _, errs := gorequest.New().
		Post("https://api.twilio.com/2010-04-01/Accounts/"+t.sid+"/Calls.json").
		SetBasicAuth(t.sid, t.token).
		Type(gorequest.TypeForm).
		Send(request).
		Timeout(10 * time.Second).
		End()

	if len(errs) > 0 {
		log.WithField("errors", errs).Error("twilio: twilio request error")
		return fmt.Errorf("twilio: can't send request to twilio")
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		bytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.WithError(err).Error("twilio: error during reading http error answer")
			return err
		}

		log.WithField("code", res.StatusCode).WithField("body", string(bytes)).Error("twilio: server responds with error code")
		return fmt.Errorf("twilio: error during sending request to twilio")
	}

	return nil
}

func (t *TwilioClient) Good(msg domain.Message) error {
	return nil
}

func (t *TwilioClient) Warn(msg domain.Message) error {
	return nil
}

func (t *TwilioClient) Report(reportId string, msg domain.Message) error {
	return nil
}

func (t *TwilioClient) Resolve(reportId string) error {
	return nil
}
