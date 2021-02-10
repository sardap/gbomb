package gbomb

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

type PodcastFeedMock struct {
}

func (c *PodcastFeedMock) Do(req *http.Request) (*http.Response, error) {
	expectedURL, _ := url.Parse("https://www.giantbomb.com/feeds/podcast/?api_key=coolbeans")

	if req.URL.String() != expectedURL.String() {
		return nil, fmt.Errorf("invalid URL %s expected %s", req.URL, expectedURL)
	}

	file, _ := os.Open("test_data/bombcast_feed.xml")

	return &http.Response{
		Body:       file,
		StatusCode: 200,
		Status:     "200",
	}, nil
}

func TestRssChannel(t *testing.T) {
	invoker := CreateInvoker("https://www.giantbomb.com", "coolbeans")
	invoker.client = &PodcastFeedMock{}
	feed, err := invoker.GetPodcasts("bombcast")
	if err != nil {
		t.Error(err)
	}

	if len(feed.Entries) != 810 {
		t.Errorf("invalid length read %d expected %d", len(feed.Entries), 810)
	}

	tme, _ := feed.Entries[0].GetPublishTime()
	expectedTme, _ := time.Parse(
		"2006-01-02 15:04:05 -0700 MST", "2021-02-09 14:52:00 +0000 PST",
	)
	if !tme.Equal(expectedTme) {
		t.Errorf(
			"didn't parse time correctly was %s expcted %s",
			tme.String(), expectedTme.String(),
		)
	}

	expctedDownload := "https://dts.podtrac.com/redirect.mp3/www.giantbomb.com/podcasts/download/3246/audio.mp3"
	if feed.Entries[0].link != expctedDownload {
		t.Errorf(
			"did not create download link correctly was %s expcted %s",
			feed.Entries[0].link, expctedDownload,
		)
	}
}
