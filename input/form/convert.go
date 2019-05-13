package form

import (
	"bytes"
	"fmt"
	"mime"
	"strings"
	"text/template"

	"github.com/pipelined/phono/input"

	"github.com/pipelined/mp3"
	"github.com/pipelined/signal"
)

// convertData provides a data for convert form, so user can define conversion parameters.
type convertData struct {
	Accept     string
	OutFormats map[string]string
	WavMime    string
	Mp3Mime    string
	WavOptions wavOptions
	Mp3Options mp3Options
	WavMaxSize int64
	Mp3MaxSize int64
	MaxSizes   map[string]int64
}

// wavOptions is a struct of wav format options that are available for conversion.
type wavOptions struct {
	BitDepths map[signal.BitDepth]struct{}
}

// mp3Options is a struct of mp3 format options that are available for conversion.
type mp3Options struct {
	ChannelModes  map[mp3.ChannelMode]struct{}
	VBR           string
	ABR           string
	CBR           string
}

const (
	fileKey = "input-file"
)

var (
	// ConvertFormData is the serialized convert form with values.
	// convertFormBytes []byte
	mp3opts = mp3Options{
		VBR:          "VBR",
		ABR:          "ABR",
		CBR:          "CBR",
		ChannelModes: mp3.Supported.ChannelModes(),
	}

	convertForm = convertData{
		WavMime:    mime.TypeByExtension(input.Wav.DefaultExtension),
		Mp3Mime:    mime.TypeByExtension(input.Mp3.DefaultExtension),
		Accept:     strings.Join(append(input.Wav.Extensions, input.Mp3.Extensions...), ", "),
		OutFormats: outFormats(input.Wav.DefaultExtension, input.Mp3.DefaultExtension),
		WavOptions: wavOptions{
			BitDepths: input.Wav.BitDepths,
		},
		Mp3Options: mp3opts,
	}

	convertTmpl = template.Must(template.New("convert").Parse(convertHTML))
)

// Convert provides user interaction via http form.
type Convert struct {
	WavMaxSize int64
	Mp3MaxSize int64
}

// Data returns serialized form data, ready to be served.
func (c Convert) Data() []byte {
	d := convertForm
	d.Mp3MaxSize = c.Mp3MaxSize
	d.WavMaxSize = c.WavMaxSize
	d.MaxSizes = maxSizes(c.WavMaxSize, c.Mp3MaxSize)

	var b bytes.Buffer
	if err := convertTmpl.Execute(&b, d); err != nil {
		panic(fmt.Sprintf("Failed to parse convert template: %v", err))
	}
	return b.Bytes()
}

func maxSizes(wavMaxSize, mp3MaxSize int64) map[string]int64 {
	m := make(map[string]int64)
	for _, ext := range input.Mp3.Extensions {
		m[ext] = mp3MaxSize
	}
	for _, ext := range input.Wav.Extensions {
		m[ext] = wavMaxSize
	}
	return m
}

// outFormats maps the extensions with values without dots.
func outFormats(exts ...string) map[string]string {
	m := make(map[string]string)
	for _, ext := range exts {
		m[ext] = ext[1:]
	}
	return m
}

type extensionsFunc func() []string

const convertHTML = `
<html>
<head>
    <style>
        * {
            font-family: Verdana;
        }
        form {
            margin: 0;
        }
        a {
            color:inherit;
        }
        button {
            background:none!important;
            color:inherit;
            border:none; 
            padding:0!important;
            font: inherit;
            border-bottom:1px solid #444; 
            cursor: pointer;
        }
        .file {
            margin-bottom: 20px;
        }
        .container {
            width: 1000px;
            margin-right: auto;
            margin-left: auto;
        }
        .outputs {
            margin-bottom: 20px;
            display: block;
        }
        .output-options {
            display: none;
        }
        .mp3-bit-rate-mode-options{
            display: none;
        }
        .mp3-quality {
            display: inline;
        }
        .option {
            margin-right: 7px;
        }
        .footer{
            position: fixed;
            padding-top: 15px;
            padding-bottom: 15px;
            bottom: 0;
        }
        #output-format-block {
            display: none;
        }
        #input-file {
            display: none;
        }
        #input-file-label {
            cursor: pointer;
            padding:0!important;
            border-bottom:1px solid #444; 
        }
    </style>
    <script type="text/javascript">
        const fileId = 'input-file';
        const accept = '{{ .Accept }}';
        function getFile() {
            return document.getElementById(fileId);
        }
        function getFileName(file) {
            var filePath = file.value;
            return filePath.substr(filePath.lastIndexOf('\\') + 1);
        }
        function getFileExtension(fileName) {
            return '.'.concat(fileName.split('.')[1]);
        }
        function humanFileSize(size) {
            var i = size == 0 ? 0 : Math.floor(Math.log(size) / Math.log(1024));
            return (size / Math.pow(1024, i)).toFixed(2) * 1 + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
        };
        function displayClass(className, mode) {
            var elements = document.getElementsByClassName(className);
            for (var i = 0, ii = elements.length; i < ii; i++) {
                elements[i].style.display = mode;
            };
        }
        function displayId(id, mode){
            document.getElementById(id).style.display = mode;
        }
        document.addEventListener('DOMContentLoaded', function(event) {
            document.getElementById('convert').reset();
            // base form handlers
            document.getElementById('input-file').addEventListener('change', onInputFileChange);
            document.getElementById('output-format').addEventListener('change', onOutputFormatChange);
            document.getElementById('submit-button').addEventListener('click', onSubmitClick);
            // mp3 handlers
            document.getElementById('mp3-bit-rate-mode').addEventListener('click', onMp3BitRateModeChange);
            document.getElementById('mp3-use-quality').addEventListener('click', onMp3UseQUalityChange);
        });
        function onInputFileChange(){
            var fileName = getFileName(getFile());
            document.getElementById('input-file-label').innerHTML = fileName;
            var ext = getFileExtension(fileName);
            if (accept.indexOf(ext) < 0) {
                alert('Only files with following extensions are allowed: {{.Accept}}')
                return;
            }
            displayClass('input-file-label', 'inline');
            displayId('output-format-block', 'inline');
        }
		function onOutputFormatChange(){
            displayClass('output-options', 'none');
            // need to cut the dot
        	displayId(this.value.slice(1)+'-options', 'inline');
        	displayClass('submit', 'block');
        }
        function onMp3BitRateModeChange(){
        	displayClass('mp3-bit-rate-mode-options', 'none');
        	var selectedOptions = 'mp3-'+this.options[this.selectedIndex].id+'-options';
        	displayClass(selectedOptions, 'inline');
        }
        function onMp3UseQUalityChange(){
            if (this.checked) {
                document.getElementById('mp3-quality-value').style.visibility = '';
            } else {
                document.getElementById('mp3-quality-value').style.visibility = 'hidden';
            }
        }
        function onSubmitClick(){
            var convert = document.getElementById('convert');
            var file = getFile();
            var ext = getFileExtension(getFileName(file));
            convert.action = ext;
            var size = file.files[0].size;
            switch (ext) {
            {{ range $ext, $maxSize := .MaxSizes }}
            case '{{$ext}}': 
                if ({{ $maxSize }} > 0 && {{ $maxSize }} < size) {
                    alert('File is too big. Maximum allowed size: '.concat(humanFileSize({{ $maxSize }})))
                    return;
                } 
                break;
            {{ end }}
            }  
            convert.submit();  
        }
    </script> 
</head>
<body>
    <div class="container">
        <h2>phono convert</h1>
        <form id="convert" enctype="multipart/form-data" method="post">
        <div class="file">
            <input id="input-file" type="file" name="input-file" accept="{{.Accept}}"/>
            <label id="input-file-label" for="input-file">select file</label>
        </div>
        <div class="outputs">
            <div id="output-format-block" class="option">
                format 
                <select id="output-format" name="format">
                    <option hidden disabled selected value>select</option>
                    {{range $key, $value := .OutFormats}}
                        <option id="{{ $value }}" value="{{ $key }}">{{ $value }}</option>
                    {{end}}
                </select>
            </div>
            <div id="wav-options" class="output-options">
                bit depth
                <select name="wav-bit-depth" class="option">
                    <option hidden disabled selected value>select</option>
                    {{range $key, $value := .WavOptions.BitDepths}}
                        <option value="{{ printf "%d" $key }}">{{ $key }}</option>
                    {{end}}
                </select>
            </div>
            <div id="mp3-options" class="output-options">
                channel mode
                <select name="mp3-channel-mode" class="option">
                    <option hidden disabled selected value>select</option>
                    {{range $key, $value := .Mp3Options.ChannelModes}}
                        <option value="{{ printf "%d" $key }}">{{ $key }}</option>
                    {{end}}
                </select>
                bit rate mode
                <select id="mp3-bit-rate-mode" class="option" name="mp3-bit-rate-mode">
                    <option hidden disabled selected value>select</option>
                    <option id="{{ .Mp3Options.VBR  }}" value="{{ .Mp3Options.VBR }}">{{ .Mp3Options.VBR }}</option>
                    <option id="{{ .Mp3Options.CBR  }}" value="{{ .Mp3Options.CBR }}">{{ .Mp3Options.CBR }}</option>
                    <option id="{{ .Mp3Options.ABR  }}" value="{{ .Mp3Options.ABR }}">{{ .Mp3Options.ABR }}</option>
                </select>
                <div class="mp3-bit-rate-mode-options mp3-{{ .Mp3Options.ABR }}-options mp3-{{ .Mp3Options.CBR }}-options">
                    bit rate [8-320]
                    <input type="text" class="option" name="mp3-bit-rate" maxlength="3" size="3">
                </div> 
                <div class="mp3-bit-rate-mode-options mp3-{{ .Mp3Options.VBR }}-options">
                    vbr quality [0-9]
                    <input type="text" class="option" name="mp3-vbr-quality" maxlength="1" size="3">
                </div>
                <div class="mp3-quality">
                    <input type="checkbox" id="mp3-use-quality" name="mp3-use-quality" value="true">quality
                    <div id="mp3-quality-value" class="mp3-quality" style="visibility:hidden">
                        [0-9]
                        <input type="text" class="option" name="mp3-quality" maxlength="1" size="3">
                    </div> 
                </div>
            </div>
        </div>
        </form>
        <div class="submit" style="display:none">
            <button id="submit-button" type="button">convert</button> 
        </div>
        <div class="footer">
            <div class="container">
            made with <a href="https://github.com/pipelined/pipe" target="_blank">pipe</a>
            </div>
        </div>
    </div>
</body>
</html>
`
