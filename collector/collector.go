package collector

import (
	"bytes"
	"errors"
	"os/exec"
	"sync"

	"github.com/sirupsen/logrus"
)

type Collector interface {
	Refresh() error
	ScriptVrrps() ([]VRRPScript, error)
	DataVrrps() (map[string]*VRRPData, error)
	StatsVrrps() (map[string]*VRRPStats, error)
	JSONVrrps() ([]VRRP, error)
	HasVRRPScriptStateSupport() bool
}

// KeepalivedCollector
type KeepalivedCollector struct {
	sync.Mutex
	useJSON    bool
	scriptPath string
	collector  Collector
}

// VRRPStats represents Keepalived stats about VRRP.
type VRRPStats struct {
	AdvertRcvd        int `json:"advert_rcvd"`
	AdvertSent        int `json:"advert_sent"`
	BecomeMaster      int `json:"become_master"`
	ReleaseMaster     int `json:"release_master"`
	PacketLenErr      int `json:"packet_len_err"`
	AdvertIntervalErr int `json:"advert_interval_err"`
	IPTTLErr          int `json:"ip_ttl_err"`
	InvalidTypeRcvd   int `json:"invalid_type_rcvd"`
	AddrListErr       int `json:"addr_list_err"`
	InvalidAuthType   int `json:"invalid_authtype"`
	AuthTypeMismatch  int `json:"authtype_mismatch"`
	AuthFailure       int `json:"auth_failure"`
	PRIZeroRcvd       int `json:"pri_zero_rcvd"`
	PRIZeroSent       int `json:"pri_zero_sent"`
}

// VRRPData represents Keepalived data about VRRP.
type VRRPData struct {
	IName        string   `json:"iname"`
	State        int      `json:"state"`
	WantState    int      `json:"wantstate"`
	Intf         string   `json:"ifp_ifname"`
	GArpDelay    int      `json:"garp_delay"`
	VRID         int      `json:"vrid"`
	VIPs         []string `json:"vips"`
	ExcludedVIPs []string `json:"excluded_vips"`
}

// VRRPScript represents Keepalived script about VRRP.
type VRRPScript struct {
	Name   string
	Status string
	State  string
}

// VRRP ties together VRRPData and VRRPStats.
type VRRP struct {
	Data  VRRPData  `json:"data"`
	Stats VRRPStats `json:"stats"`
}

// KeepalivedStats ties together VRRP and VRRPScript.
type KeepalivedStats struct {
	VRRPs   []VRRP
	Scripts []VRRPScript
}

// NewKeepalivedCollector is creating new instance of KeepalivedCollector.
func NewKeepalivedCollector(useJSON bool, scriptPath string, collector Collector) *KeepalivedCollector {
	kc := &KeepalivedCollector{
		useJSON:    useJSON,
		scriptPath: scriptPath,
		collector:  collector,
	}

	return kc
}

func (k *KeepalivedCollector) GetKeepalivedStats() (*KeepalivedStats, error) {
	stats := &KeepalivedStats{
		VRRPs:   make([]VRRP, 0),
		Scripts: make([]VRRPScript, 0),
	}

	var err error

	if err := k.collector.Refresh(); err != nil {
		return nil, err
	}

	if k.useJSON {
		stats.VRRPs, err = k.collector.JSONVrrps()
		if err != nil {
			return nil, err
		}

		return stats, nil
	}

	stats.Scripts, err = k.collector.ScriptVrrps()
	if err != nil {
		return nil, err
	}

	vrrpStats, err := k.collector.StatsVrrps()
	if err != nil {
		return nil, err
	}

	vrrpData, err := k.collector.DataVrrps()
	if err != nil {
		return nil, err
	}

	if len(vrrpData) != len(vrrpStats) {
		logrus.Error("keepalived.data and keepalived.stats datas are not synced")

		return nil, errors.New("keepalived.data and keepalived.stats datas are not synced")
	}

	for instance, vData := range vrrpData {
		if vStat, ok := vrrpStats[instance]; ok {
			stats.VRRPs = append(stats.VRRPs, VRRP{
				Data:  *vData,
				Stats: *vStat,
			})
		} else {
			logrus.WithField("instance", instance).Error("There is no stats found for instance")

			return nil, errors.New("there is no stats found for instance")
		}
	}

	return stats, nil
}

func (k *KeepalivedCollector) checkScript(vip string) bool {
	var stdout, stderr bytes.Buffer

	script := k.scriptPath + " " + vip
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{"VIP": vip, "stdout": stdout.String(), "stderr": stderr.String()}).
			WithError(err).
			Error("Check script failed")

		return false
	}

	return true
}
