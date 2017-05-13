package smuxnet

import (
	"testing"

	"encoding/json"

	"time"

	"sync"

	"math"

	"bytes"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zenhotels/chanserv"
)

type req struct {
	Sources int
	Frames  int
	Frame   string
}

func TestOk(t *testing.T) {
	size := uint64(math.MaxUint32) * 2
	buf := bytes.NewBuffer(make([]byte, size))
	buf.WriteString("a")
	fmt.Println("buf done", size, buf.Len())
	return

	r := req{
		Sources: 1,
		Frames:  1,
		Frame:   buf.String(),
	}
	cli := setupServerAndCli(t)
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go sendReq(t, r, cli, wg)
	}
	wg.Wait()
}

func sendReq(t *testing.T, r req, cli Client, wg *sync.WaitGroup) {
	defer wg.Done()
	body, err := json.Marshal(&r)
	require.NoError(t, err)

	out, errs2 := cli.Request(body, time.Now().Add(time.Hour))
	go func() {
		for err := range errs2 {
			require.NoError(t, err)
		}
	}()

	counter := 0
	for frame := range out {
		counter++
		require.Equal(t, len(frame.Bytes()), len(r.Frame))
	}
	require.Equal(t, r.Frames*r.Sources, counter)
}

func setupServerAndCli(t *testing.T) Client {
	srcFunc := func(body []byte) <-chan chanserv.Source {
		out := make(chan chanserv.Source)
		go func() {
			var r req
			assert.NoError(t, json.Unmarshal(body, &r))
			for i := 0; i < r.Sources; i++ {
				frames := make([]frame, r.Frames)
				for j := 0; j < r.Frames; j++ {
					frames[j] = frame(r.Frame)
				}
				src := &source{frames: frames, out: make(chan chanserv.Frame)}
				go src.writeFrames(t)
				out <- src
			}
			close(out)
		}()
		return out
	}

	srv, _ := NewServer(0, 0, 100)
	errs := srv.ListenAndServe(":9001", srcFunc)
	go func() {
		for err := range errs {
			require.NoError(t, err)
		}
	}()
	cli, err := NewClient("test", "tcp4", ":9001", 0, 0, 100)
	require.NoError(t, err)
	return cli
}

type source struct {
	frames []frame
	header []byte
	out    chan chanserv.Frame
}

func (s *source) Header() []byte {
	return s.header
}

func (s *source) Meta() chanserv.MetaData {
	return nil
}

func (s *source) Out() <-chan chanserv.Frame {
	return s.out
}

func (s *source) writeFrames(t *testing.T) {
	for _, frame := range s.frames {
		s.out <- &frame
	}
	close(s.out)
}
