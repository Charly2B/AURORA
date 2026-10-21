package probe

import "testing"

const sample = `{
 "streams": [
  {"index":0,"codec_type":"video","codec_name":"mjpeg","disposition":{"attached_pic":1}},
  {"index":1,"codec_type":"video","codec_name":"hevc","profile":"Main 10","pix_fmt":"yuv420p10le","width":3840,"height":2160,
   "avg_frame_rate":"24000/1001","color_transfer":"smpte2084",
   "side_data_list":[{"side_data_type":"DOVI configuration record","dv_profile":8}]},
  {"index":2,"codec_type":"audio","codec_name":"truehd","profile":"Dolby TrueHD + Dolby Atmos","channels":8,"tags":{"language":"eng"},"disposition":{"default":1}},
  {"index":3,"codec_type":"audio","codec_name":"ac3","channels":6,"tags":{"language":"fre","title":"VFF"}},
  {"index":4,"codec_type":"subtitle","codec_name":"hdmv_pgs_subtitle","tags":{"language":"fre"},"disposition":{"forced":1}},
  {"index":5,"codec_type":"subtitle","codec_name":"subrip","tags":{"language":"und"}}
 ],
 "chapters":[{"start_time":"0.000000","end_time":"300.5","tags":{"title":"Ouverture"}}],
 "format":{"format_name":"matroska,webm","duration":"7200.5","bit_rate":"52000000"}
}`

func TestParse(t *testing.T) {
	r, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if r.VideoStream != 1 || r.VideoCodec != "hevc" || r.Height != 2160 {
		t.Errorf("flux vidéo : la pochette intégrée doit être ignorée (%+v)", r)
	}
	if r.HDR != "Dolby Vision + HDR10" {
		t.Errorf("HDR = %q", r.HDR)
	}
	if r.FrameRate < 23.97 || r.FrameRate > 23.98 {
		t.Errorf("cadence = %v", r.FrameRate)
	}
	if len(r.Audio) != 2 || !r.Audio[0].Atmos || r.Audio[0].Stream != 2 || r.Audio[1].Index != 1 || r.Audio[1].Language != "fre" {
		t.Errorf("audio = %+v", r.Audio)
	}
	if len(r.Subtitles) != 2 || r.Subtitles[0].Text || !r.Subtitles[0].Forced || !r.Subtitles[1].Text || r.Subtitles[1].Language != "" {
		t.Errorf("sous-titres = %+v", r.Subtitles)
	}
	if len(r.Chapters) != 1 || r.Chapters[0].Title != "Ouverture" || r.Duration != 7200.5 {
		t.Errorf("chapitres/durée = %+v %v", r.Chapters, r.Duration)
	}
}

func TestHDRNotClaimedWithoutEvidence(t *testing.T) {
	r, _ := Parse([]byte(`{"streams":[{"index":0,"codec_type":"video","codec_name":"hevc","width":3840,"height":2160}],"format":{"duration":"10"}}`))
	if r.HDR != "" {
		t.Errorf("aucun HDR ne doit être déclaré sans métadonnée de transfert : %q", r.HDR)
	}
	if _, err := Parse([]byte(`{"streams":[],"format":{}}`)); err == nil {
		t.Error("un fichier sans flux ni durée doit être signalé en erreur")
	}
}
