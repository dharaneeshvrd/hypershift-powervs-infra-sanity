package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/openshift/hypershift/cmd/infra/powervs"
	"github.com/openshift/hypershift/cmd/log"
)

/*
var powerVsRegionZoneM = map[string][]string{
	"osa":      {"osa21"},
	"us-south": {"us-south"},
	"dal":      {"dal12"},
	"eu-de":    {"eu-de-2"},
	"tor":      {"tor01"},
	"sao":      {"sao01"},
	"lon":      {"lon04"},
	"syd":      {"syd04"},
	"tok":      {"tok04"},
	"us-east":  {"us-east"},
	"wdc":      {"wdc06"},
}*/

var powerVsRegionZoneM = map[string][]string{
	"osa":      {"osa21"},
	"us-south": {"us-south"},
	"dal":      {"dal12"},
	"eu-de":    {"eu-de-2"},
	"tor":      {"tor01"},
	"sao":      {"sao01"},
	"lon":      {"lon04"},
	"syd":      {"syd04"},
	"us-east":  {"us-east"},
	"wdc":      {"wdc06"},
}

var powerVsRegionL = []string{"osa", "us-south", "dal", "eu-de", "tor", "sao", "lon", "syd", "tok", "us-east", "wdc"}

var vpcRegionL = []string{"jp-osa", "us-south", "us-south", "eu-de", "ca-tor", "br-sao", "eu-gb", "au-syd", "jp-tok", "us-east", "us-east"}

const (
	infraNamePrefix = "hyp-sanity"
)

var m sync.Mutex

type sanityResult struct {
	infra   powervs.Infra
	options powervs.CreateInfraOptions
}

type sanityResultL []sanityResult

var sanityResults = make(sanityResultL, 0)

func runSanity(options powervs.CreateInfraOptions, wg *sync.WaitGroup) {
	log.Log.WithName(options.InfraID).Info("runSanity called with", "options", options)
	infra := &powervs.Infra{ID: options.InfraID}

	err := infra.SetupInfra(&options)

	if err != nil {
		log.Log.WithName(options.InfraID).Error(err, "sanity failed")
	}

	m.Lock()
	sanityResults = append(sanityResults, sanityResult{infra: *infra, options: options})
	m.Unlock()

	wg.Done()
}

func cleanInfra(options powervs.CreateInfraOptions, wg *sync.WaitGroup) {
	log.Log.WithName(options.InfraID).Info("cleanInfra called with", "options", options)
	destroyOptions := powervs.DestroyInfraOptions{InfraID: options.InfraID,
		ResourceGroup: options.ResourceGroup,
		PowerVSRegion: options.PowerVSRegion,
		PowerVSZone:   options.PowerVSZone,
		VpcRegion:     options.VpcRegion,
	}
	infra := &powervs.Infra{}
	err := destroyOptions.DestroyInfra(infra)
	if err != nil {
		log.Log.WithName(options.InfraID).Info("error cleaing up", "infra", options.InfraID, "err", err)
	}
	wg.Done()
}

func main() {
	args := os.Args[1:]

	if len(args) <= 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Println("Usage: ./infra_sanity <baseDomain> <resourceGroup> <mode - [all, one]>")
		return
	}

	baseDomain := args[0]
	resourceGroup := args[1]
	mode := args[2]

	var wg sync.WaitGroup

	for index, region := range powerVsRegionL {
		for _, zone := range powerVsRegionZoneM[region] {

			vpcRegion := vpcRegionL[index]
			options := powervs.CreateInfraOptions{BaseDomain: baseDomain,
				Debug:         true,
				ResourceGroup: resourceGroup,
				PowerVSRegion: region,
				PowerVSZone:   zone,
				VpcRegion:     vpcRegion,
			}

			infraID := fmt.Sprintf("%s-%s", infraNamePrefix, zone)
			options.InfraID = infraID

			log.Log.WithName(infraID).Info("triggering sanity for", "powervsRegion", region, "powervsZone", zone, "vpcRegion", vpcRegion)
			go runSanity(options, &wg)
			wg.Add(1)

			//time.Sleep(time.Second * 10)
			if mode == "one" {
				break
			}
		}
		if mode == "one" {
			break
		}
	}

	wg.Wait()

	outFile := "sanity_results.json"
	var jsonOut = make(map[string]interface{})

	for _, result := range sanityResults {
		out := map[string]interface{}{"infra": result.infra}
		jsonOut[result.options.InfraID] = out
	}

	var err error
	out, err := os.Create(outFile)
	if err != nil {
		log.Log.WithName("sanity results").Error(err, "error creating results file")
	}
	defer out.Close()

	outputBytes, err := json.MarshalIndent(jsonOut, "", "  ")
	if err != nil {
		log.Log.WithName("sanity results").Error(err, "failed to serialize results")
	}
	_, err = out.Write(outputBytes)
	if err != nil {
		log.Log.WithName("sanity results").Error(err, "failed to write results json")
	}

	for _, result := range sanityResults {
		go cleanInfra(result.options, &wg)
		wg.Add(1)
		log.Log.WithName("sanity results").Info("triggered cleaning", "infraID", result.options.InfraID)
	}

	wg.Wait()
}
