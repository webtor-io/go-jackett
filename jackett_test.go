package jackett

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	testJackett *Jackett
)

const (
	testAPIKey string = "abracadabra"
)

func TestGenerateURL(t *testing.T) {
	tests := []struct {
		input *FetchRequest
		want  string
	}{
		{&FetchRequest{}, "/api/v2.0/indexers/all/results?apikey=abracadabra"},
		{&FetchRequest{
			Trackers:   []string{"aaa", "bbb"},
			Categories: []uint{1, 2},
			Query:      "qqq",
		}, "/api/v2.0/indexers/all/results?Category%5B%5D=1&Category%5B%5D=2&Query=qqq&Tracker%5B%5D=aaa&Tracker%5B%5D=bbb&apikey=abracadabra"},
	}
	for _, test := range tests {
		got, err := testJackett.generateFetchURL(test.input)
		if err != nil {
			log.Fatal(err)
		}
		if !strings.HasSuffix(got, test.want) {
			t.Errorf("strings.HasSuffix(generateURL(%+v), %q), want %q", test.input, got, test.want)
		}
	}
}

func TestFetch(t *testing.T) {
	ctx := context.Background()
	input := &FetchRequest{}
	got, err := testJackett.Fetch(ctx, input)
	if err != nil {
		log.Fatal(err)
	}
	if len(got.Indexers) != 5 {
		t.Errorf("len(Fetch(%+v).Indexers) = %v, want 5", input, len(got.Indexers))
	}
	if len(got.Results) != 3 {
		t.Errorf("len(Fetch(%+v).Results) = %v, want 3", input, len(got.Results))
	}
}

func init() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Results":[`))
		w.Write([]byte(`{"FirstSeen":"0001-01-01T00:00:00","Tracker":"1337x","TrackerId":"1337x","CategoryDesc":"Audio","BlackholeLink":null,"Title":"RUSH Oakland Coliseum Oakland CA 1992 01 30 Speed Corrected ASM vers of Mirrors","Guid":"https://1337x.to/torrent/4524786/RUSH-Oakland-Coliseum-Oakland-CA-1992-01-30-Speed-Corrected-ASM-vers-of-Mirrors/","Link":"https://jackett.example.xyz:443/dl/1337x/?jackett_apikey=***&path=Q2ZESjhGLWF3Y2pLMDlsQ3JJZnhWM1dUUUw3akZrUHdVTlhUVTc5eHQ4VFd2NVJud2lBcDR4TllKb0pYV3ZzdFNaRHR1OFFVazl2ME9tX2c5eHVMTTJLQ0hVeVRoY3JXNEJwazNFdnhkSzZwX0oyUkpSLUlzZmltX1RtOWNsY0Z4eFNPaVhZbmp2d3U5emVfVWFXZDhlaUNoaHFpb2RTYUhzOUFjTG05enJnR1BIaXQ5c2Z2MU8zR2k3MDNhMG8wckFOUE83RG5TTHVkQVVibjA5MHRReWU2dWtsNGxTVkNFYVZoU1dIWG9JcURUNVU0T0xTMDlZZk5sZ0xrWGkwV1FOem5WMjBfZmdMTktNbWYwbHNrTFliRl9pcw&file=RUSH+Oakland+Coliseum+Oakland+CA+1992+01+30+Speed+Corrected+ASM+vers+of+Mirrors","Comments":"https://1337x.to/torrent/4524786/RUSH-Oakland-Coliseum-Oakland-CA-1992-01-30-Speed-Corrected-ASM-vers-of-Mirrors/","PublishDate":"2020-07-01T12:39:19.3915488+01:00","Category":[3000,100068],"Size":883635008,"Files":91,"Grabs":null,"Description":null,"RageID":null,"TVDBId":null,"Imdb":null,"TMDb":null,"Seeders":0,"Peers":0,"BannerUrl":null,"InfoHash":null,"MagnetUri":null,"MinimumRatio":1.0,"MinimumSeedTime":172800,"DownloadVolumeFactor":0.0,"UploadVolumeFactor":1.0,"Gain":0.0},`))
		w.Write([]byte(`{"FirstSeen":"0001-01-01T00:00:00","Tracker":"MyCornClub","TrackerId":"mycornclub","CategoryDesc":"XXX","BlackholeLink":null,"Title":"Grandpa With Son Baking Aunt And Sons Girlfriend","Guid":"https://mycorn.club/torrent/e68TkEko","Link":"https://jackett.example.xyz:443/dl/mycornclub/?jackett_apikey=***&path=Q2ZESjhGLWF3Y2pLMDlsQ3JJZnhWM1dUUUw3YU1TRER6MlZmdzBBSm0wRlVUbFdNeUxuMXNqTm1wWlJXWTlhcnJmV1BxQlF3eFJDTGlBeEdaZU9hQmFUYzBtV2ZINmlCUm9WY3BDdHV1djVSTmZlbEtoU3AwdDhLMGpzWnZxVHcweEhKTml6ek94bjRtdGJWQ3RhSmdpbVpKNktxcXBZZ1JQRWpXNzV5VTllYm9sQzI&file=Grandpa+With+Son+Baking+Aunt+And+Sons+Girlfriend","Comments":"https://mycorn.club/torrent/e68TkEko","PublishDate":"2020-07-01T12:30:14.9551777+01:00","Category":[6000],"Size":537741248,"Files":null,"Grabs":2,"Description":null,"RageID":null,"TVDBId":null,"Imdb":null,"TMDb":null,"Seeders":0,"Peers":0,"BannerUrl":null,"InfoHash":null,"MagnetUri":null,"MinimumRatio":1.0,"MinimumSeedTime":172800,"DownloadVolumeFactor":0.0,"UploadVolumeFactor":1.0,"Gain":0.0},`))
		w.Write([]byte(`{"FirstSeen":"0001-01-01T00:00:00","Tracker":"RARBG","TrackerId":"rarbg","CategoryDesc":"TV/HD","BlackholeLink":null,"Title":"Gardeners.World.S53E12.720p.HDTV.x264-dotTV[rartv]","Guid":"magnet:?xt=urn:btih:828d3f022e50c0d038e64b7d2c981645a812ce2b&dn=Gardeners.World.S53E12.720p.HDTV.x264-dotTV%5Brartv%5D&tr=http%3A%2F%2Ftracker.trackerfix.com%3A80%2Fannounce&tr=udp%3A%2F%2F9.rarbg.me%3A2710&tr=udp%3A%2F%2F9.rarbg.to%3A2710&tr=udp%3A%2F%2Fopen.demonii.com%3A1337%2Fannounce","Link":null,"Comments":"https://torrentapi.org/redirect_to_info.php?token=glws9u2tyk&p=2_2_5_8_1_0_7__828d3f022e&app_id=jackett_v0.16.783.0","PublishDate":"2020-07-01T12:28:35+01:00","Category":[5040,100041],"Size":1204300287,"Files":null,"Grabs":null,"Description":null,"RageID":null,"TVDBId":82623,"Imdb":260618,"TMDb":15188,"Seeders":10,"Peers":6,"BannerUrl":null,"InfoHash":"828d3f022e50c0d038e64b7d2c981645a812ce2b","MagnetUri":"magnet:?xt=urn:btih:828d3f022e50c0d038e64b7d2c981645a812ce2b&dn=Gardeners.World.S53E12.720p.HDTV.x264-dotTV%5Brartv%5D&tr=http%3A%2F%2Ftracker.trackerfix.com%3A80%2Fannounce&tr=udp%3A%2F%2F9.rarbg.me%3A2710&tr=udp%3A%2F%2F9.rarbg.to%3A2710&tr=udp%3A%2F%2Fopen.demonii.com%3A1337%2Fannounce","MinimumRatio":1.0,"MinimumSeedTime":172800,"DownloadVolumeFactor":0.0,"UploadVolumeFactor":1.0,"Gain":11.215920438989997}`))
		w.Write([]byte(`],"Indexers":[`))
		w.Write([]byte(`{"ID":"1337x","Name":"1337x","Status":2,"Results":60,"Error":null},`))
		w.Write([]byte(`{"ID":"mycornclub","Name":"MyCornClub","Status":2,"Results":40,"Error":null},`))
		w.Write([]byte(`{"ID":"onejav","Name":"OneJAV","Status":2,"Results":20,"Error":null},`))
		w.Write([]byte(`{"ID":"cornleech","Name":"CornLeech","Status":1,"Results":0,"Error":"Jackett.Common.IndexerException: ..."},`))
		w.Write([]byte(`{"ID":"rarbg","Name":"RARBG","Status":2,"Results":100,"Error":null}`))
		w.Write([]byte(`]}`))
	}))
	testJackett = NewJackett(&Settings{server.URL, testAPIKey, server.Client()})
}
