package proxmox

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	pxapi "github.com/Telmate/proxmox-api-go/proxmox"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// using a global variable here so that we have an internally accessible
// way to look into our own resource definition. Useful for dynamically doing typecasts
// so that we can print (debug) our ResourceData constructs
var thisResource *schema.Resource

func resourceDataToFlatValues(d *schema.ResourceData, resource *schema.Resource) (map[string]interface{}, error) {

	flatValues := make(map[string]interface{})

	for key, value := range resource.Schema {
		switch value.Type {
		case schema.TypeString:
			flatValues[key] = d.Get(key).(string)
		case schema.TypeBool:
			flatValues[key] = d.Get(key).(bool)
		case schema.TypeInt:
			flatValues[key] = d.Get(key).(int)
		case schema.TypeFloat:
			flatValues[key] = d.Get(key).(float64)
		case schema.TypeSet:
			values, _ := schemaSetToFlatValues(d.Get(key).(*schema.Set), value.Elem.(*schema.Resource))
			flatValues[key] = values
		case schema.TypeList:
			values, _ := schemaListToFlatValues(d.Get(key).([]interface{}), value.Elem.(*schema.Resource))
			flatValues[key] = values
		default:
			flatValues[key] = "? Print Not Implemented ?"
		}
	}

	return flatValues, nil
}

func schemaSetToFlatValues(set *schema.Set, resource *schema.Resource) ([]map[string]interface{}, error) {

	flatValues := make([]map[string]interface{}, 0, 1)

	for _, set := range set.List() {
		innerFlatValues := make(map[string]interface{})

		setAsMap := set.(map[string]interface{})
		for key, value := range resource.Schema {
			switch value.Type {
			case schema.TypeString:
				innerFlatValues[key] = setAsMap[key].(string)
			case schema.TypeBool:
				innerFlatValues[key] = setAsMap[key].(bool)
			case schema.TypeInt:
				innerFlatValues[key] = setAsMap[key].(int)
			case schema.TypeFloat:
				innerFlatValues[key] = setAsMap[key].(float64)
			default:
				innerFlatValues[key] = "? Print Not Implemented ?"
			}
		}

		flatValues = append(flatValues, innerFlatValues)
	}
	return flatValues, nil
}

func schemaListToFlatValues(schemaList []interface{}, resource *schema.Resource) ([]map[string]interface{}, error) {

	flatValues := make([]map[string]interface{}, 0, 1)

	for _, item := range schemaList {
		innerFlatValues := make(map[string]interface{})

		itemAsMap := item.(map[string]interface{})
		for key, value := range resource.Schema {
			switch value.Type {
			case schema.TypeString:
				innerFlatValues[key] = itemAsMap[key].(string)
			case schema.TypeBool:
				innerFlatValues[key] = itemAsMap[key].(bool)
			case schema.TypeInt:
				innerFlatValues[key] = itemAsMap[key].(int)
			case schema.TypeFloat:
				innerFlatValues[key] = itemAsMap[key].(float64)
			default:
				innerFlatValues[key] = "? Print Not Implemented ?"
			}
		}

		flatValues = append(flatValues, innerFlatValues)
	}
	return flatValues, nil
}

func resourceVmQemu() *schema.Resource {

	*pxapi.Debug = true
	thisResource = &schema.Resource{
		Create: resourceVmQemuCreate,
		Read:   resourceVmQemuRead,
		Update: resourceVmQemuUpdate,
		Delete: resourceVmQemuDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vmid": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"define_connection_info": { // by default define SSH for provisioner info
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"desc": {
				Type:     schema.TypeString,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"target_node": {
				Type:     schema.TypeString,
				Required: true,
			},
			"bios": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "seabios",
			},
			"onboot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"boot": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "cdn",
			},
			"bootdisk": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"agent": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"iso": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"clone": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"full_clone": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  true,
			},
			"hastate": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"qemu_os": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "l26",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "l26" {
						return len(d.Get("clone").(string)) > 0 // the cloned source may have a different os, which we shoud leave alone
					}
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"memory": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  512,
			},
			"balloon": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"cores": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"sockets": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"vcpus": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"cpu": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "host",
			},
			"numa": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"kvm": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"hotplug": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "network,disk,usb",
			},
			"scsihw": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vga": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "std",
						},
						"memory": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"network": &schema.Schema{
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"nic", "bridge", "vlan", "mac"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						//"id": &schema.Schema{
						//	Type:     schema.TypeInt,
						//	Deprecated:  "Id is no longer required. The order of the network blocks is used for the Id.",
						//	Optional: true,
						//},
						"model": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"macaddr": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"bridge": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "nat",
						},
						"tag": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "VLAN tag.",
							Default:     -1,
						},
						"firewall": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"rate": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"queues": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"link_down": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"disk": &schema.Schema{
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"disk_gb", "storage", "storage_type"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"storage": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"storage_type": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "dir",
							Description: "One of PVE types as described: https://pve.proxmox.com/wiki/Storage",
						},
						"size": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"format": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "raw",
						},
						"cache": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "none",
						},
						"backup": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"iothread": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"replicate": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						//SSD emulation
						"ssd": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"discard": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(string)
								if !strings.Contains(v, "ignore") && !strings.Contains(v, "on") {
									errs = append(errs, fmt.Errorf("%q, must be 'ignore'(default) or 'on', got %s", key, v))
								}
								return
							},
						},
						//Maximum r/w speed in megabytes per second
						"mbps": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"mbps_rd": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"mbps_rd_max": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"mbps_wr": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"mbps_wr_max": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			// Deprecated single disk config.
			"disk_gb": {
				Type:       schema.TypeFloat,
				Deprecated: "Use `disk.size` instead",
				Optional:   true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// bigger ok
					oldf, _ := strconv.ParseFloat(old, 64)
					newf, _ := strconv.ParseFloat(new, 64)
					return oldf >= newf
				},
			},
			"storage": {
				Type:       schema.TypeString,
				Deprecated: "Use `disk.storage` instead",
				Optional:   true,
			},
			"storage_type": {
				Type:       schema.TypeString,
				Deprecated: "Use `disk.type` instead",
				Optional:   true,
				ForceNew:   false,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "" {
						return true // empty template ok
					}
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			// Deprecated single nic config.
			"nic": {
				Type:       schema.TypeString,
				Deprecated: "Use `network` instead",
				Optional:   true,
			},
			"bridge": {
				Type:       schema.TypeString,
				Deprecated: "Use `network.bridge` instead",
				Optional:   true,
			},
			"vlan": {
				Type:       schema.TypeInt,
				Deprecated: "Use `network.tag` instead",
				Optional:   true,
				Default:    -1,
			},
			"mac": {
				Type:       schema.TypeString,
				Deprecated: "Use `network.macaddr` to access the auto generated MAC address",
				Optional:   true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "" {
						return true // macaddr auto-generates and its ok
					}
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"serial": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"os_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"os_network_config": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"ssh_forward_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_user": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"ssh_private_key": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"force_create": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"clone_wait": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  15,
			},
			"ci_wait": { // how long to wait before provision
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "" {
						return true // old empty ok
					}
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"ciuser": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"cipassword": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "**********" {
						return true // api returns astericks instead of password so can't diff
					}
					return false
				},
			},
			"cicustom": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"searchdomain": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true, // could be pre-existing if we clone from a template with it defined
			},
			"nameserver": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true, // could be pre-existing if we clone from a template with it defined
			},
			"sshkeys": {
				Type:     schema.TypeString,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"ipconfig0": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"ipconfig1": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"ipconfig2": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"preprovision": {
				Type:          schema.TypeBool,
				Optional:      true,
				Default:       true,
				ConflictsWith: []string{"ssh_forward_ip", "ssh_user", "ssh_private_key", "os_type", "os_network_config"},
			},
			"pool": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_host": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ssh_port": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"force_recreate_on_change_of": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
	return thisResource
}

var rxIPconfig = regexp.MustCompile("ip6?=([0-9a-fA-F:\\.]+)")

func resourceVmQemuCreate(d *schema.ResourceData, meta interface{}) error {

	// create a logger for this function
	logger, _ := CreateSubLogger("resource_vm_create")

	// DEBUG print out the create request
	flatValue, _ := resourceDataToFlatValues(d, thisResource)
	jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Str("vmid", d.Id()).Msgf("Invoking VM create with Id '%v' and resource data:  '%+v'", string(jsonString))

	pconf := meta.(*providerConfiguration)
	pmParallelBegin(pconf)
	client := pconf.Client
	vmName := d.Get("name").(string)
	vga := d.Get("vga").(*schema.Set)
	qemuVgaList := vga.List()

	qemuNetworks, _ := ExpandDevicesList(d.Get("network").([]interface{}))
	qemuDisks, _ := ExpandDevicesList(d.Get("disk").([]interface{}))

	serials := d.Get("serial").(*schema.Set)
	qemuSerials, _ := DevicesSetToMap(serials)

	config := pxapi.ConfigQemu{
		Name:         vmName,
		Description:  d.Get("desc").(string),
		Pool:         d.Get("pool").(string),
		Bios:         d.Get("bios").(string),
		Onboot:       d.Get("onboot").(bool),
		Boot:         d.Get("boot").(string),
		BootDisk:     d.Get("bootdisk").(string),
		Agent:        d.Get("agent").(int),
		Memory:       d.Get("memory").(int),
		Balloon:      d.Get("balloon").(int),
		QemuCores:    d.Get("cores").(int),
		QemuSockets:  d.Get("sockets").(int),
		QemuVcpus:    d.Get("vcpus").(int),
		QemuCpu:      d.Get("cpu").(string),
		QemuNuma:     d.Get("numa").(bool),
		QemuKVM:      d.Get("kvm").(bool),
		Hotplug:      d.Get("hotplug").(string),
		Scsihw:       d.Get("scsihw").(string),
		HaState:      d.Get("hastate").(string),
		QemuOs:       d.Get("qemu_os").(string),
		QemuNetworks: qemuNetworks,
		QemuDisks:    qemuDisks,
		QemuSerials:  qemuSerials,
		// Cloud-init.
		CIuser:       d.Get("ciuser").(string),
		CIpassword:   d.Get("cipassword").(string),
		CIcustom:     d.Get("cicustom").(string),
		Searchdomain: d.Get("searchdomain").(string),
		Nameserver:   d.Get("nameserver").(string),
		Sshkeys:      d.Get("sshkeys").(string),
		Ipconfig0:    d.Get("ipconfig0").(string),
		Ipconfig1:    d.Get("ipconfig1").(string),
		Ipconfig2:    d.Get("ipconfig2").(string),
		// Deprecated single disk config.
		Storage:  d.Get("storage").(string),
		DiskSize: d.Get("disk_gb").(float64),
		// Deprecated single nic config.
		QemuNicModel: d.Get("nic").(string),
		QemuBrige:    d.Get("bridge").(string),
		QemuVlanTag:  d.Get("vlan").(int),
		QemuMacAddr:  d.Get("mac").(string),
	}
	if len(qemuVgaList) > 0 {
		config.QemuVga = qemuVgaList[0].(map[string]interface{})
	}
	log.Print("[DEBUG] checking for duplicate name")
	dupVmr, _ := client.GetVmRefByName(vmName)

	forceCreate := d.Get("force_create").(bool)
	targetNode := d.Get("target_node").(string)
	pool := d.Get("pool").(string)

	if dupVmr != nil && forceCreate {
		pmParallelEnd(pconf)
		return fmt.Errorf("Duplicate VM name (%s) with vmId: %d. Set force_create=false to recycle", vmName, dupVmr.VmId())
	} else if dupVmr != nil && dupVmr.Node() != targetNode {
		pmParallelEnd(pconf)
		return fmt.Errorf("Duplicate VM name (%s) with vmId: %d on different target_node=%s", vmName, dupVmr.VmId(), dupVmr.Node())
	}

	vmr := dupVmr

	if vmr == nil {
		// get unique id
		nextid, err := nextVmId(pconf)
		vmID := d.Get("vmid").(int)
		if vmID != 0 {
			nextid = vmID
		} else {
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}
		}

		vmr = pxapi.NewVmRef(nextid)
		vmr.SetNode(targetNode)
		if pool != "" {
			vmr.SetPool(pool)
		}

		// check if ISO or clone
		if d.Get("clone").(string) != "" {
			fullClone := 1
			if !d.Get("full_clone").(bool) {
				fullClone = 0
			}
			config.FullClone = &fullClone

			sourceVmr, err := client.GetVmRefByName(d.Get("clone").(string))
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}

			log.Print("[DEBUG] cloning VM")
			err = config.CloneVm(sourceVmr, vmr, client)

			if err != nil {
				pmParallelEnd(pconf)
				return err
			}

			err = config.UpdateConfig(vmr, client)
			if err != nil {
				// Set the id because when update config fail the vm is still created
				d.SetId(resourceId(targetNode, "qemu", vmr.VmId()))
				pmParallelEnd(pconf)
				return err
			}

			// give sometime to proxmox to catchup
			time.Sleep(time.Duration(d.Get("clone_wait").(int)) * time.Second)

			err = prepareDiskSize(client, vmr, qemuDisks)
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}

		} else if d.Get("iso").(string) != "" {
			config.QemuIso = d.Get("iso").(string)
			err := config.CreateVm(vmr, client)
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}
		} else {
			return fmt.Errorf("Either clone or iso must be set")
		}
	} else {
		log.Printf("[DEBUG] recycling VM vmId: %d", vmr.VmId())

		client.StopVm(vmr)

		err := config.UpdateConfig(vmr, client)
		if err != nil {
			// Set the id because when update config fail the vm is still created
			d.SetId(resourceId(targetNode, "qemu", vmr.VmId()))
			pmParallelEnd(pconf)
			return err
		}

		// give sometime to proxmox to catchup
		time.Sleep(5 * time.Second)

		err = prepareDiskSize(client, vmr, qemuDisks)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}
	}
	d.SetId(resourceId(targetNode, "qemu", vmr.VmId()))
	logger.Debug().Int("vmid", vmr.VmId()).Msgf("Set this vm (resource Id) to '%v'", d.Id())

	// give sometime to proxmox to catchup
	time.Sleep(15 * time.Second)

	log.Print("[DEBUG] starting VM")
	_, err := client.StartVm(vmr)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	err = initConnInfo(d, pconf, client, vmr, &config)
	if err != nil {
		return err
	}

	return resourceVmQemuRead(d, meta)
}

func resourceVmQemuUpdate(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)

	// create a logger for this function
	logger, _ := CreateSubLogger("resource_vm_update")

	pmParallelBegin(pconf)
	client := pconf.Client
	_, _, vmID, err := parseResourceId(d.Id())
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	logger.Info().Int("vmid", vmID).Msg("Starting update of the VM resource")

	vmr := pxapi.NewVmRef(vmID)
	_, err = client.GetVmInfo(vmr)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}
	vga := d.Get("vga").(*schema.Set)
	qemuVgaList := vga.List()

	qemuDisks, _ := ExpandDevicesList(d.Get("disk").([]interface{}))
	qemuNetworks, err := ExpandDevicesList(d.Get("network").([]interface{}))
	if err != nil {
		return fmt.Errorf("Error while processing Network configuration: %v", err)
	}
	logger.Debug().Int("vmid", vmID).Msgf("Processed NetworkSet into qemuNetworks as %+v", qemuNetworks)

	serials := d.Get("serial").(*schema.Set)
	qemuSerials, _ := DevicesSetToMap(serials)

	d.Partial(true)
	if d.HasChange("target_node") {
		_, err := client.MigrateNode(vmr, d.Get("target_node").(string), true)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}
		d.SetPartial("target_node")
		vmr.SetNode(d.Get("target_node").(string))
	}
	d.Partial(false)

	config := pxapi.ConfigQemu{
		Name:         d.Get("name").(string),
		Description:  d.Get("desc").(string),
		Pool:         d.Get("pool").(string),
		Bios:         d.Get("bios").(string),
		Onboot:       d.Get("onboot").(bool),
		Boot:         d.Get("boot").(string),
		BootDisk:     d.Get("bootdisk").(string),
		Agent:        d.Get("agent").(int),
		Memory:       d.Get("memory").(int),
		Balloon:      d.Get("balloon").(int),
		QemuCores:    d.Get("cores").(int),
		QemuSockets:  d.Get("sockets").(int),
		QemuVcpus:    d.Get("vcpus").(int),
		QemuCpu:      d.Get("cpu").(string),
		QemuNuma:     d.Get("numa").(bool),
		QemuKVM:      d.Get("kvm").(bool),
		Hotplug:      d.Get("hotplug").(string),
		Scsihw:       d.Get("scsihw").(string),
		HaState:      d.Get("hastate").(string),
		QemuOs:       d.Get("qemu_os").(string),
		QemuNetworks: qemuNetworks,
		QemuDisks:    qemuDisks,
		QemuSerials:  qemuSerials,
		// Cloud-init.
		CIuser:       d.Get("ciuser").(string),
		CIpassword:   d.Get("cipassword").(string),
		CIcustom:     d.Get("cicustom").(string),
		Searchdomain: d.Get("searchdomain").(string),
		Nameserver:   d.Get("nameserver").(string),
		Sshkeys:      d.Get("sshkeys").(string),
		Ipconfig0:    d.Get("ipconfig0").(string),
		Ipconfig1:    d.Get("ipconfig1").(string),
		Ipconfig2:    d.Get("ipconfig2").(string),
		// Deprecated single disk config.
		Storage:  d.Get("storage").(string),
		DiskSize: d.Get("disk_gb").(float64),
		// Deprecated single nic config.
		QemuNicModel: d.Get("nic").(string),
		QemuBrige:    d.Get("bridge").(string),
		QemuVlanTag:  d.Get("vlan").(int),
		QemuMacAddr:  d.Get("mac").(string),
	}
	if len(qemuVgaList) > 0 {
		config.QemuVga = qemuVgaList[0].(map[string]interface{})
	}

	logger.Debug().Int("vmid", vmID).Msgf("Updating VM with the following configuration: %+v", config)

	err = config.UpdateConfig(vmr, client)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	// give sometime to proxmox to catchup
	time.Sleep(5 * time.Second)

	prepareDiskSize(client, vmr, qemuDisks)

	// give sometime to proxmox to catchup
	time.Sleep(15 * time.Second)

	// Start VM only if it wasn't running.
	vmState, err := client.GetVmState(vmr)
	if err == nil && vmState["status"] == "stopped" {
		log.Print("[DEBUG] starting VM")
		_, err = client.StartVm(vmr)
	} else if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	err = initConnInfo(d, pconf, client, vmr, &config)
	if err != nil {
		return err
	}

	// give sometime to bootup
	time.Sleep(9 * time.Second)
	if _, err = client.StopVm(vmr); err != nil {
		pmParallelEnd(pconf)
		return err
	}

	time.Sleep(9 * time.Second)
	if _, err = client.StartVm(vmr); err != nil {
		pmParallelEnd(pconf)
		return err
	}

	return resourceVmQemuRead(d, meta)
}

func resourceVmQemuRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	pmParallelBegin(pconf)
	client := pconf.Client

	_, _, vmID, err := parseResourceId(d.Id())
	if err != nil {
		pmParallelEnd(pconf)
		d.SetId("")
		return fmt.Errorf("Unexpected error when trying to read and parse the resource: %v", err)
	}

	// create a logger for this function
	logger, _ := CreateSubLogger("resource_vm_read")
	logger.Info().Int("vmid", vmID).Msg("Reading configuration for vmid")

	vmr := pxapi.NewVmRef(vmID)

	// Try to get information on the vm. If this call err's out
	// that indicates the VM does not exist. We indicate that to terraform
	// by calling a SetId("")
	_, err = client.GetVmInfo(vmr)
	if err != nil {
		pmParallelEnd(pconf)
		d.SetId("")
		return nil
	}

	config, err := pxapi.NewConfigQemuFromApi(vmr, client)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	logger.Debug().Int("vmid", vmID).Msgf("[READ] Received Config from Proxmox API: %+v", config)

	d.SetId(resourceId(vmr.Node(), "qemu", vmr.VmId()))
	d.Set("target_node", vmr.Node())
	d.Set("name", config.Name)
	d.Set("desc", config.Description)
	d.Set("pool", config.Pool)
	d.Set("bios", config.Bios)
	d.Set("onboot", config.Onboot)
	d.Set("boot", config.Boot)
	d.Set("bootdisk", config.BootDisk)
	d.Set("agent", config.Agent)
	d.Set("memory", config.Memory)
	d.Set("balloon", config.Balloon)
	d.Set("cores", config.QemuCores)
	d.Set("sockets", config.QemuSockets)
	d.Set("vcpus", config.QemuVcpus)
	d.Set("cpu", config.QemuCpu)
	d.Set("numa", config.QemuNuma)
	d.Set("kvm", config.QemuKVM)
	d.Set("hotplug", config.Hotplug)
	d.Set("scsihw", config.Scsihw)
	d.Set("hastate", vmr.HaState())
	d.Set("qemu_os", config.QemuOs)
	// Cloud-init.
	d.Set("ciuser", config.CIuser)
	// we purposely use the password from the terraform config here
	// because the proxmox api will always return "**********" leading to diff issues
	d.Set("cipassword", d.Get("cipassword").(string))
	d.Set("cicustom", config.CIcustom)
	d.Set("searchdomain", config.Searchdomain)
	d.Set("nameserver", config.Nameserver)
	d.Set("sshkeys", config.Sshkeys)
	d.Set("ipconfig0", config.Ipconfig0)
	d.Set("ipconfig1", config.Ipconfig1)
	d.Set("ipconfig2", config.Ipconfig2)

	// Disks.
	// add an explicit check that the keys in the config.QemuDisks map are a strict subset of
	// the keys in our resource schema. if they aren't things fail in a very weird and hidden way
	for _, diskEntry := range config.QemuDisks {
		for key, _ := range diskEntry {
			if _, ok := thisResource.Schema["disk"].Elem.(*schema.Resource).Schema[key]; !ok {
				if key == "id" { // we purposely ignore id here as that is implied by the order in the TypeList/QemuDevice(list)
					continue
				}
				return fmt.Errorf("Proxmox Provider Error: proxmox API returned new disk parameter '%v' we cannot process", key)
			}
		}
	}
	flatDisks, _ := FlattenDevicesList(config.QemuNetworks)
	d.Set("disk", flatDisks)

	// Display.
	activeVgaSet := d.Get("vga").(*schema.Set)
	if len(activeVgaSet.List()) > 0 {
		d.Set("features", UpdateDeviceConfDefaults(config.QemuVga, activeVgaSet))
	}

	// Networks.
	// add an explicit check that the keys in the config.QemuNetworks map are a strict subset of
	// the keys in our resource schema. if they aren't things fail in a very weird and hidden way
	logger.Debug().Int("vmid", vmID).Msgf("Network block received '%v'", d.Get("network"))
	for _, networkEntry := range config.QemuNetworks {
		for key, _ := range networkEntry {
			if _, ok := thisResource.Schema["network"].Elem.(*schema.Resource).Schema[key]; !ok {
				if key == "id" { // we purposely ignore id here as that is implied by the order in the TypeList/QemuDevice(list)
					continue
				}
				return fmt.Errorf("Proxmox Provider Error: proxmox API returned new network parameter '%v' we cannot process", key)
			}
		}
	}
	flatNetworks, _ := FlattenDevicesList(config.QemuNetworks)
	d.Set("network", flatNetworks)

	// Deprecated single disk config.
	d.Set("storage", config.Storage)
	d.Set("disk_gb", config.DiskSize)
	d.Set("storage_type", config.StorageType)
	// Deprecated single nic config.
	d.Set("nic", config.QemuNicModel)
	d.Set("bridge", config.QemuBrige)
	d.Set("vlan", config.QemuVlanTag)
	d.Set("mac", config.QemuMacAddr)
	d.Set("pool", vmr.Pool())
	//Serials
	configSerialsSet := d.Get("serial").(*schema.Set)
	activeSerialSet := UpdateDevicesSet(configSerialsSet, config.QemuSerials)
	d.Set("serial", activeSerialSet)

	pmParallelEnd(pconf)

	// DEBUG print out the read result
	flatValue, _ := resourceDataToFlatValues(d, thisResource)
	jsonString, _ := json.Marshal(flatValue)
	logger.Debug().Int("vmid", vmID).Msgf("Finished VM read resulting in data: '%+v'", string(jsonString))

	return nil
}

func resourceVmQemuDelete(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	pmParallelBegin(pconf)
	client := pconf.Client
	vmId, _ := strconv.Atoi(path.Base(d.Id()))
	vmr := pxapi.NewVmRef(vmId)
	_, err := client.StopVm(vmr)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}
	// give sometime to proxmox to catchup
	time.Sleep(2 * time.Second)
	_, err = client.DeleteVm(vmr)
	pmParallelEnd(pconf)
	return err
}

// Increase disk size if original disk was smaller than new disk.
func prepareDiskSize(
	client *pxapi.Client,
	vmr *pxapi.VmRef,
	diskConfMap pxapi.QemuDevices,
) error {
	clonedConfig, err := pxapi.NewConfigQemuFromApi(vmr, client)
	if err != nil {
		return err
	}
	//log.Printf("%s", clonedConfig)
	for diskID, diskConf := range diskConfMap {
		diskName := fmt.Sprintf("%v%v", diskConf["type"], diskID)

		diskSize := diskSizeGB(diskConf["size"])

		if _, diskExists := clonedConfig.QemuDisks[diskID]; !diskExists {
			return err
		}

		clonedDiskSize := diskSizeGB(clonedConfig.QemuDisks[diskID]["size"])

		if err != nil {
			return err
		}

		diffSize := int(math.Ceil(diskSize - clonedDiskSize))
		if diskSize > clonedDiskSize {
			log.Print("[DEBUG] resizing disk " + diskName)
			_, err = client.ResizeQemuDisk(vmr, diskName, diffSize)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func diskSizeGB(dcSize interface{}) float64 {
	var diskSize float64
	switch dcSize.(type) {
	case string:
		diskString := strings.ToUpper(dcSize.(string))
		re := regexp.MustCompile("([0-9]+)([A-Z]*)")
		diskArray := re.FindStringSubmatch(diskString)

		diskSize, _ = strconv.ParseFloat(diskArray[1], 64)

		if len(diskArray) >= 3 {
			switch diskArray[2] {
			case "G", "GB":
				//Nothing to do
			case "M", "MB":
				diskSize /= 1000
			case "K", "KB":
				diskSize /= 1000000
			}
		}
	case float64:
		diskSize = dcSize.(float64)
	}
	return diskSize
}

// Converting from schema.TypeSet to map of id and conf for each device,
// which will be sent to Proxmox API.
func DevicesSetToMap(devicesSet *schema.Set) (pxapi.QemuDevices, error) {

	var err error
	devicesMap := pxapi.QemuDevices{}

	for _, set := range devicesSet.List() {
		setMap, isMap := set.(map[string]interface{})
		if isMap {
			setID := setMap["id"].(int)
			if _, ok := devicesMap[setID]; !ok {
				devicesMap[setID] = setMap
			} else {
				return nil, fmt.Errorf("Unable to process set, received a duplicate ID '%v' check your configuration file", setID)
			}
		}
	}
	return devicesMap, err
}

// Consumes an API return (pxapi.QemuDevices) and "flattens" it into a []map[string]interface{} as
// expected by the terraform interface for TypeList
func FlattenDevicesList(proxmoxDevices pxapi.QemuDevices) ([]map[string]interface{}, error) {
	flattenedDevices := make([]map[string]interface{}, 0, 1)

	numDevices := len(proxmoxDevices)
	if numDevices == 0 {
		return flattenedDevices, nil
	}

	// QemuDevices is a map[int]map[string]interface{}
	// we loop by index here to ensure that the devices remain in the same order
	for index := 0; index < numDevices; index++ {
		thisDevice := proxmoxDevices[index]
		thisFlattenedDevice := make(map[string]interface{})

		if thisDevice == nil {
			continue
		}

		for configuration, value := range thisDevice {
			thisFlattenedDevice[configuration] = value
		}

		flattenedDevices = append(flattenedDevices, thisFlattenedDevice)
	}

	return flattenedDevices, nil
}

// Consumes a terraform TypeList of a Qemu Device (network, hard drive, etc) and returns the "Expanded"
// version of the equivalent configuration that the API understands (the struct pxapi.QemuDevices).
// NOTE this expects the provided deviceList to be []map[string]interface{}.
func ExpandDevicesList(deviceList []interface{}) (pxapi.QemuDevices, error) {
	expandedDevices := make(pxapi.QemuDevices)

	if len(deviceList) == 0 {
		return expandedDevices, nil
	}

	for index, deviceInterface := range deviceList {
		thisDeviceMap := deviceInterface.(map[string]interface{})

		// allocate an expandedDevice, we'll append it to the list at the end of this loop
		thisExpandedDevice := make(map[string]interface{})

		// bail out if the device is empty, it is meaningless in this context
		if thisDeviceMap == nil {
			continue
		}

		// this is a map of string->interface, loop over it and move it into
		// the qemuDevices struct
		for configuration, value := range thisDeviceMap {
			thisExpandedDevice[configuration] = value
		}

		expandedDevices[index] = thisExpandedDevice
	}

	return expandedDevices, nil
}

// Update schema.TypeSet with new values comes from Proxmox API.
// TODO: remove these set functions and convert attributes using a set to a list instead.
func UpdateDevicesSet(
	devicesSet *schema.Set,
	devicesMap pxapi.QemuDevices,
) *schema.Set {

	//configDevicesMap, _ := DevicesSetToMap(devicesSet)

	//activeDevicesMap := updateDevicesDefaults(devicesMap, configDevicesMap)
	activeDevicesMap := devicesMap

	for _, setConf := range devicesSet.List() {
		devicesSet.Remove(setConf)
		setConfMap := setConf.(map[string]interface{})
		deviceID := setConfMap["id"].(int)
		// Value type should be one of types allowed by Terraform schema types.
		for key, value := range activeDevicesMap[deviceID] {
			// This nested switch is used for nested config like in `net[n]`,
			// where Proxmox uses `key=<0|1>` in string" at the same time
			// a boolean could be used in ".tf" files.
			switch setConfMap[key].(type) {
			case bool:
				switch value.(type) {
				// If the key is bool and value is int (which comes from Proxmox API),
				// should be converted to bool (as in ".tf" conf).
				case int:
					sValue := strconv.Itoa(value.(int))
					bValue, err := strconv.ParseBool(sValue)
					if err == nil {
						setConfMap[key] = bValue
					}
				// If value is bool, which comes from Terraform conf, add it directly.
				case bool:
					setConfMap[key] = value
				}
			// Anything else will be added as it is.
			default:
				setConfMap[key] = value
			}
		}
		devicesSet.Add(setConfMap)
	}

	return devicesSet
}

// Because default values are not stored in Proxmox, so the API returns only active values.
// So to prevent Terraform doing unnecessary diffs, this function reads default values
// from Terraform itself, and fill empty fields.
func updateDevicesDefaults(
	activeDevicesMap pxapi.QemuDevices,
	configDevicesMap pxapi.QemuDevices,
) pxapi.QemuDevices {

	for deviceID, deviceConf := range configDevicesMap {
		if _, ok := activeDevicesMap[deviceID]; !ok {
			activeDevicesMap[deviceID] = configDevicesMap[deviceID]
		}
		for key, value := range deviceConf {
			if _, ok := activeDevicesMap[deviceID][key]; !ok {
				activeDevicesMap[deviceID][key] = value
			}
		}
	}
	return activeDevicesMap
}

func initConnInfo(
	d *schema.ResourceData,
	pconf *providerConfiguration,
	client *pxapi.Client,
	vmr *pxapi.VmRef,
	config *pxapi.ConfigQemu) error {

	// allow user to opt-out of setting the connection info for the resource
	if !d.Get("define_connection_info").(bool) {
		return nil
	}

	sshPort := "22"
	sshHost := ""
	var err error
	if config.HasCloudInit() {
		if d.Get("ssh_forward_ip") != nil {
			sshHost = d.Get("ssh_forward_ip").(string)
		}
		if sshHost == "" {
			if d.Get("ipconfig0").(string) == "ip=dhcp" {
				guestAgentSupported := false
				guestAgentRunning := false
				// look if this vm has set the qemu guest agent flag
				vmState, err := client.GetVmState(vmr)
				if err != nil {
					return err
				}
				if vmState["agent"] != nil && vmState["agent"].(float64) == 1 {
					// the chances are good that the vm will run a guest agent
					guestAgentSupported = true
				}
				// wait until the os has started the guest agent
				for end := time.Now().Add(60 * time.Second); guestAgentSupported; {
					_, err := client.GetVmAgentNetworkInterfaces(vmr)
					if err == nil {
						guestAgentRunning = true
						break
					} else if !strings.Contains(err.Error(), "QEMU guest agent is not running") {
						// "not running" means either not installed or not started yet.
						// any other error should not happen here
						return err
					}
					if time.Now().After(end) {
						break
					}
					time.Sleep(10 * time.Second)
				}
				if guestAgentRunning {
					// wait until we find a valid ipv4 address
					for end := time.Now().Add(60 * time.Second); guestAgentSupported; {
						ifs, err := client.GetVmAgentNetworkInterfaces(vmr)
						if err != nil {
							return err
						}
						for _, iface := range ifs {
							for _, addr := range iface.IPAddresses {
								if addr.IsGlobalUnicast() && strings.Count(addr.String(), ":") < 2 {
									sshHost = addr.String()
									break
								}
							}
							if sshHost != "" {
								break
							}
						}
						if time.Now().After(end) || sshHost != "" {
							break
						}
						time.Sleep(10 * time.Second)
					}
				}
			} else {
				// parse IP address out of ipconfig0
				ipMatch := rxIPconfig.FindStringSubmatch(d.Get("ipconfig0").(string))
				sshHost = ipMatch[1]
			}
		}
		// Check if we got a speficied port
		if strings.Contains(sshHost, ":") {
			sshParts := strings.Split(sshHost, ":")
			sshHost = sshParts[0]
			sshPort = sshParts[1]
		}
	} else {
		log.Print("[DEBUG] setting up SSH forward")
		sshPort, err = pxapi.SshForwardUsernet(vmr, client)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}
		sshHost = d.Get("ssh_forward_ip").(string)
	}

	// Done with proxmox API, end parallel and do the SSH things
	pmParallelEnd(pconf)

	// Optional convience attributes for provisioners
	d.Set("ssh_host", sshHost)
	d.Set("ssh_port", sshPort)

	// This connection INFO is longer shared up to the providers :-(
	d.SetConnInfo(map[string]string{
		"type":            "ssh",
		"host":            sshHost,
		"port":            sshPort,
		"user":            d.Get("ssh_user").(string),
		"private_key":     d.Get("ssh_private_key").(string),
		"pm_api_url":      client.ApiUrl,
		"pm_user":         client.Username,
		"pm_password":     client.Password,
		"pm_otp":          client.Otp,
		"pm_tls_insecure": "true", // TODO - pass pm_tls_insecure state around, but if we made it this far, default insecure
	})
	return nil
}
