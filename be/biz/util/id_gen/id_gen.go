package id_gen

import (
	"doing_now/be/biz/util/ip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/gopkg/lang/fastrand"
)

func init() {
	idgen = NewIDGenerator(10)
}

func NewID() string {
	return idgen.NewID()
}

var idgen *IDGenerator

type IDGenerator struct {
	pool <-chan string
	stop chan any
}

func NewIDGenerator(maxSize int) *IDGenerator {
	stop := make(chan any)
	idgen := &IDGenerator{
		pool: newPool(maxSize, stop),
		stop: stop,
	}

	return idgen
}

func (idgen *IDGenerator) Stop() {
	select {
	case <-idgen.stop:
	default:
		close(idgen.stop)
	}
}

func (idgen *IDGenerator) NewID() string {
	return <-idgen.pool
}

func newPool(size int, stop chan any) <-chan string {
	pool := make(chan string, size)

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				sb := strings.Builder{}
				sb.WriteString(strconv.FormatUint(uint64(time.Now().UnixMilli()), 36))
				sb.WriteString(ip.IPv4Hex())
				sb.WriteString(strconv.FormatUint(uint64(os.Getpid()), 10))
				sb.WriteString(strconv.FormatUint(fastrand.Uint64(), 36))

				pool <- sb.String()
			}
		}
	}()

	return pool
}
