package gbomb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const giantBombTimeFormat = "2006-01-02 15:04:05"

//Invoker Invoker
type Invoker struct {
	Endpoint string
	APIKey   string
	limter   *rate.Limiter
}

func (i *Invoker) requestLimiter() {
	i.limter.Wait(context.TODO())
}

func (i *Invoker) get(pageable Pageable) ([]byte, error) {
	url := fmt.Sprintf("%s/%s", i.Endpoint, pageable.Path())

	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", url, nil,
	)
	if err != nil {
		return nil, err
	}

	offset, err := pageable.NextOffset()
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", i.APIKey)
	q.Add("format", "json")
	q.Add("offset", fmt.Sprintf("%d", offset))
	req.URL.RawQuery = q.Encode()

	i.requestLimiter()
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

//CreateInvoker Creates a giant bomb invoker
func CreateInvoker(endpoint, key string) *Invoker {
	return &Invoker{
		Endpoint: endpoint, APIKey: key,
		limter: rate.NewLimiter(rate.Every(time.Duration(31)*time.Second), 1),
	}
}

//Pageable used for reuqests with many pages
type Pageable interface {
	NextOffset() (int, error)
	Complete() bool
	Path() string
}

//Image a giant bomb API Image
type Image struct {
	IconURL        string `json:"icon_url"`
	MediumURL      string `json:"medium_url"`
	ScreenURL      string `json:"screen_url"`
	ScreenLargeURL string `json:"screen_large_url"`
	SmallURL       string `json:"small_url"`
	SuperURL       string `json:"super_url"`
	ThumbURL       string `json:"thumb_url"`
	TinyURL        string `json:"tiny_url"`
	OriginalURL    string `json:"original_url"`
	ImageTags      string `json:"image_tags"`
}

//VideoShow Giant bomb api VideoShow
type VideoShow struct {
	APIDetailURL  string `json:"api_detail_url"`
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Postion       int    `json:"postion"`
	SiteDetailURL string `json:"site_detail_url"`
	Image         Image  `json:"image"`
	Logo          Image  `json:"logo"`
}

//Association an Association
type Association struct {
	APIDetailURL  string `json:"api_detail_url"`
	SiteDetailURL string `json:"site_detail_url"`
	GUID          string `json:"guid"`
	ID            string `json:"id"`
	Name          string `json:"name"`
}

//VideoCategory a VideoCategory
type VideoCategory struct {
	APIDetailURL  string `json:"api_detail_url"`
	SiteDetailURL string `json:"site_detail_url"`
	ID            string `json:"id"`
	Name          string `json:"name"`
}

//VideoInfo a VideoInfo
type VideoInfo struct {
	DetailURL       string          `json:"api_detail_url"`
	SiteDetailURL   string          `json:"site_detail_url"`
	GUID            string          `json:"guid"`
	ID              string          `json:"id"`
	Associations    []Association   `json:"associations"`
	Deck            string          `json:"deck"`
	EmbedPlayer     string          `json:"embed_player"`
	LengthSeconds   int             `json:"length_seconds"`
	Name            string          `json:"name"`
	Premium         bool            `json:"premium"`
	PublishDate     string          `json:"publish_date"`
	User            string          `json:"user"`
	Hosts           string          `json:"Hosts"`
	Crew            string          `json:"crew"`
	VideoType       string          `json:"video_type"`
	Show            VideoShow       `json:"video_show"`
	VideoCategories []VideoCategory `json:"video_categories"`
	SavedTime       string          `json:"saved_time"`
	YoutubeID       string          `json:"youtube_id"`
	LowURL          string          `json:"low_url"`
	HighURL         string          `json:"high_url"`
	HDURL           string          `json:"hd_url"`
	URL             string          `json:"url"`
}

//LengthDuration Returns the legnth as time.Duration
func (v *VideoInfo) LengthDuration() time.Duration {
	return time.Duration(v.LengthSeconds) * time.Second
}

//OnYoutube returns if video can be found on youtube
func (v *VideoInfo) OnYoutube() bool {
	return v.YoutubeID != ""
}

//GetHighestURL returns high URL if valid or low URL
func (v *VideoInfo) GetHighestURL() string {
	if v.HighURL != "" {
		return v.HighURL
	}

	return v.LowURL
}

//GetBestQuailtyURL returns the best Quailty video url
func (v *VideoInfo) GetBestQuailtyURL() string {
	if v.HDURL != "" {
		return v.HDURL
	}

	return v.GetHighestURL()
}

//PublishDateTime will return publish date as Time
func (v *VideoInfo) PublishDateTime() time.Time {
	res, _ := time.Parse(giantBombTimeFormat, v.PublishDate)
	return res
}

//ResponsePage the page part of a response
type ResponsePage struct {
	Error       string `json:"error"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	PageResults int    `json:"number_of_page_results"`
	MaxResults  int    `json:"number_of_total_results"`
	StatusCode  int    `json:"status_code"`
}

//NextOffset returns the next offset
func (r *ResponsePage) NextOffset() (int, error) {
	//First call
	if r.PageResults == 0 {
		return r.Offset, nil
	}

	if r.Offset < r.MaxResults {
		next := r.MaxResults - r.Offset
		if next > r.Limit {
			next = r.Limit
		}

		r.Offset += next

		return r.Offset, nil
	}

	return 0, fmt.Errorf("no more results")
}

//Complete will return if there are no more results
func (r *ResponsePage) Complete() bool {
	return r.Offset >= r.MaxResults
}

//VideosResponse videos response from giant bomb API
type VideosResponse struct {
	ResponsePage
	Videos []VideoInfo `json:"results"`
}

//Path returns video path
func (v *VideosResponse) Path() string {
	return fmt.Sprintf("api/videos")
}

//Next returns next page for video response
func (v *VideosResponse) Next(i *Invoker) error {
	body, err := i.get(v)
	if err != nil {
		return err
	}

	var tmp VideosResponse
	json.Unmarshal(body, &tmp)

	v.ResponsePage = tmp.ResponsePage
	v.Videos = tmp.Videos

	return nil
}

//GetVideos returns a base video
func (i *Invoker) GetVideos(ctx context.Context, offset int) (*VideosResponse, error) {
	result := &VideosResponse{}
	result.Offset = offset
	result.ResponsePage.Limit = 100
	result.MaxResults = 100

	body, err := i.get(result)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(body, result)

	return result, nil
}

//DownloadVideo downloads a given video
func (i *Invoker) DownloadVideo(ctx context.Context, url string) (io.ReadCloser, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("api_key", i.APIKey)
	req.URL.RawQuery = q.Encode()

	i.requestLimiter()
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}
