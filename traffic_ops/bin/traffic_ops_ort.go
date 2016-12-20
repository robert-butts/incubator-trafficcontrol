// +build amd64,linux

package main

import (
	"math/rand"
	"os/exec"
	// "bytes"
	// "crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	// "net/http"
	// "net/url"
	to "github.com/apache/incubator-trafficcontrol/traffic_ops/client"
	"golang.org/x/sys/unix"
	"os"
	"os/user"
	"strings"
	"time"
)

type Mode string

const ModeInteractive = "interactive"
const ModeReport = "report"
const ModeBadass = "badass"
const ModeSyncds = "syncds"
const ModeInvalid = ""

func ModeFromStr(s string) Mode {
	switch strings.ToLower(s) {
	case "interactive":
		return ModeInteractive
	case "report":
		return ModeReport
	case "badass":
		return ModeBadass
	case "syncds":
		return ModeSyncds
	default:
		return ModeInvalid
	}
}

func (m Mode) String() string {
	return string(m)
}

type LogLevel int

const LogLevelAll = 8
const LogLevelTrace = 7
const LogLevelDebug = 6
const LogLevelInfo = 5
const LogLevelWarn = 4
const LogLevelError = 3
const LogLevelFatal = 2
const LogLevelNone = 1
const LogLevelInvalid = 0

func LogLevelFromStr(s string) LogLevel {
	switch strings.ToLower(s) {
	case "all":
		return LogLevelAll
	case "trace":
		return LogLevelTrace
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarn
	case "error":
		return LogLevelError
	case "fatal":
		return LogLevelFatal
	case "none":
		return LogLevelNone
	default:
		return LogLevelInvalid
	}
}

func (e LogLevel) String() string {
	switch e {
	case LogLevelAll:
		return "all"
	case LogLevelTrace:
		return "trace"
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelFatal:
		return "fatal"
	case LogLevelNone:
		return "none"
	default:
		return "invalid"
	}
}

func (p Params) Log(level LogLevel, msg string) {
	if p.LogLevel < level {
		return
	}
	fmt.Printf("%s", msg)
}

func (p Params) Logn(level LogLevel, msg string) {
	p.Log(level, fmt.Sprintf("%s: %s\n", strings.ToUpper(level.String()), msg))
}

const DefaultDispersion = 300
const DefaultRetries = 5
const DefaultWaitForParents = true

var Dispersion = flag.Int("dispersion", DefaultDispersion, "wait a random number between 0 and <time> before starting")
var Retries = flag.Int("retries", DefaultRetries, "retry connection to Traffic Ops URL <number> times")
var WaitForParents = flag.Bool("wait_for_parents", DefaultWaitForParents, "do not update if parent_pending = 1 in the update json")

type Params struct {
	Mode
	LogLevel
	TrafficOpsHost string
	TrafficOpsUser string
	TrafficOpsPass string
	Dispersion     time.Duration
	WaitForParents bool
}

func getParams() (Params, error) {
	if len(os.Args) < 5 {
		return Params{}, fmt.Errorf("insufficient arguments")
	}

	params := Params{}

	params.Mode = ModeFromStr(os.Args[1])
	if params.Mode == ModeInvalid {
		return Params{}, fmt.Errorf("invalid mode")
	}

	params.LogLevel = LogLevelFromStr(os.Args[2])
	if params.LogLevel == LogLevelInvalid {
		return Params{}, fmt.Errorf("invalid log level")
	}

	params.TrafficOpsHost = os.Args[3]
	if !(strings.HasPrefix(params.TrafficOpsHost, "http://") || strings.HasPrefix(params.TrafficOpsHost, "https://")) {
		return Params{}, fmt.Errorf("invalid traffic ops host")
	}

	login := os.Args[4]
	userPass := strings.Split(login, ":")
	if len(userPass) < 2 {
		return Params{}, fmt.Errorf("invalid login")
	}
	params.TrafficOpsUser = userPass[0]
	params.TrafficOpsPass = userPass[1]
	params.Dispersion = time.Second * time.Duration(*Dispersion)
	params.WaitForParents = WaitForParents
	return params, nil
}

const TmpBase = "/tmp/ort"

const YumOpts = ""
const TsHome = "/opt/trafficserver"
const TrafficLine = TsHome + "/bin/traffic_line"

type ConfigFileInfo struct {
	RemapPluginConfigFile string
	ChangeApplied         bool
	Contents              string
	AuditComplete         bool
	Location              string
	ChangeNeeded          bool
	Service               string
	PrereqFailed          bool
	BackupFromTrafficOps  string
	FnameInTrafficOps     string
	BackupFromDisk        string
	Component             string
}

type SSLInfo struct {
	CertName string
	KeyName  string
}

type UpdateStatusTrafficOps string

const UpdateStatusTrafficOpsNeeded = UpdateStatusTrafficOps("needed")
const UpdateStatusTrafficOpsSuccessful = UpdateStatusTrafficOps("successful")
const UpdateStatusTrafficOpsFailed = UpdateStatusTrafficOps("failed")
const UpdateStatusTrafficOpsNotNeeded = UpdateStatusTrafficOps("") // default condition

// TODO make local
var installTracker = map[string]struct{}{}

func main() {
	fmt.Printf("%s\n", time.Now().Format("Mon Jan 2 15:04:05 MST 2006"))

	params, err := getParams()
	if err != nil {
		fmt.Printf("Error getting params: %v\n", err)
		printUsage()
		return
	}

	release, err := osVersion()
	if err != nil {
		fmt.Printf("error getting OS version: %v\n", err)
		return
	}
	release = strings.ToUpper(release)
	params.Logn(LogLevelDebug, "OS release is "+release)
	if !SupportedRelease(release) {
		fmt.Printf("unsupported el_version: %v\n", release)
		return
	}

	lockedFile, err := checkOnlyCopyRunning()
	if err != nil {
		params.Logn(LogLevelFatal, os.Args[0]+" is already running. Exiting.\n")
		return
	}
	defer lockedFile.Close()

	unixTime := time.Now().Unix()

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Error getting hostname: %v\n", err)
		return
	}
	hostnameShort := hostNameShort(hostname)
	domainName := domainName(hostname)
	userAgent := fmt.Sprintf("%v-%v", hostnameShort, unixTime)

	err = checkRunUser(params)
	if err != nil {
		params.Logn(LogLevelFatal, err.Error())
		return
	}
	// client := &http.Client{}
	// req.Header.Set("User-Agent", userAgent)

	fmt.Printf("hostname %v domainname %v useragent %v\n", hostnameShort, domainName, userAgent)

	toSession, err := to.Login(params.TrafficOpsHost, params.TrafficOpsUser, params.TrafficOpsPass, true)
	if err != nil {
		params.Logn(LogLevelFatal, "Error logging into Traffic Ops: "+err.Error())
		return
	}

	// DEBUG
	cdns, err := toSession.CDNs()
	if err != nil {
		params.Logn(LogLevelFatal, "Error logging into Traffic Ops req: "+err.Error())
		return
	}
	fmt.Printf("CDNs: %+v\n", cdns)
	// for {
	// 	time.Sleep(100)
	// }
	params.Logn(LogLevelDebug, "YUM_OPTS: "+YumOpts+".")

	out, err := run("/usr/bin/yum clean metadata")
	checkOutput(params, out, err)

	// DEBUG
	fmt.Printf("output: %v\n", out)

	// rebootNeeded := false
	// trafficLineNeeded := false
	// sysctlPNeeded := false
	// ntpdRestartNeeded := false
	// atsRunning := false
	// // teakdRunning := false
	// installedNewSslKeys := false
	// installTracker := map[string]struct{}{}
	// configFileTracker := map[string]ConfigFileInfo{}
	// sslTracker := map[string]SSLInfo{}

	// Start main flow

	// First and foremost, if this is a syncds run, check to see if we can bail.

	syncdsUpdate := checkSyncdsState(params, toSession, hostnameShort)
	if syncdsUpdate == UpdateStatusTrafficOpsNotNeeded {
		return // TODO fix - the pl exits immediately from checkSyncdsState
	}
	fmt.Printf("%v\n", syncdsUpdate) // debug

	deleteOldTmpDirs(params)

	profileName, cfgFileTracker, cdnName, err := getCfgFileList(hostnameShort, toSession)
	if err != nil {
		params.Logn(LogLevelFatal, "Error getting config file list from Traffic Ops: "+err.Error())
		return
	}

	headerComment := getHeaderComment(toSession)
	if !processPackages(params, hostnameShort, toSession) {
		return // processPackages itself logs, so we don't here
	}

	// get the ats user's UID after package installation in case this is the initial badass
	atsUid := getpwnam("ats")
	processChkconfig(hostnameShort, toSession)

}

func processChkconfig(hostname string, toClient *to.Session) {

}

// processPackages returns whether to continue, or exit. It logs appropriately, so if false is returned, callers can exit without further messages.
func processPackages(params Params, hostname string, toClient *to.Session) bool {
	packages, err := toClient.Packages(hostname) //	my $url     = "$tm_host/ort/$host_name/packages";
	if err != nil {
		params.Logn(LogLevelFatal, "Error getting package list from Traffic Ops! "+err.Error())
		return false
	}

	proceed := false
	installPkgs := map[string]struct{}{}   // package_map{"install"}
	uninstallPkgs := map[string]struct{}{} //package_map{"uninstall"}

	//		iterate through to build the uninstall list
	for _, pkg := range packages {
		fullPkg := getFullPkgName(pkg.Name, pkg.Version)
		for _, installedPkg := range pkgInstalled(pkg.Name) {
			if _, ok := uninstallPkgs[fullPkg]; ok {
				params.Logn(LogLevelInfo, fullPkg+": Already marked for removal.")
				continue // TODo should this be checking installedPkg, not fullPkg?
				// ( $log_level >> $INFO ) && print "INFO $full_package: Already marked for removal.\n";
			} else if installedPkg == fullPkg {
				params.Logn(LogLevelInfo, fullPkg+": Currently installed and not marked for removal.")
				continue
			}

			if params.Mode == ModeReport {
				params.Logn(LogLevelFatal, "ERROR: "+installedPkg+": Currently installed and needs to be removed.")
			} else {
				params.Logn(LogLevelTrace, installedPkg+": Currently installed, marked for removal.")
			}

			uninstallPkgs[installedPkg] = struct{}{}

			// add any dependent packages to the list of packages to uninstall
			// TODO determine if this should pass fullPkg
			for _, dependentPkg := range pkgRequires(pkg.Name) {
				if params.Mode == ModeReport {
					params.Logn(LogLevelFatal, "ERROR "+dependentPkg+": Currently installed and depends on "+pkg.Name+" and needs to be removed.")
				} else {
					params.Logn(LogLevelTrace, dependentPkg+": Currently installed and depends on "+pkg.Name+", marked for removal.")
				}
				uninstallPkgs[dependentPkg] = struct{}{}
			}
		}
	}

	// iterate through to build the install list
	// TODO combine loops? Extract method?
	for _, pkg := range packages {
		fullPkg := getFullPkgName(pkg.Name, pkg.Version)
		if len(pkgInstalled(pkg.Name, pkg.Version)) == 0 {
			if params.Mode == ModeReport {
				params.Logn(LogLevelFatal, "ERROR "+fullPkg+": Needs to be installed.")
			} else {
				params.Logn(LogLevelTrace, fullPkg+": Needs to be installed.")
			}
			installPkgs[fullPkg] = struct{}{}
		} else if _, ok := uninstallPkgs[fullPkg]; ok {
			if params.Mode == ModeReport {
				params.Logn(LogLevelFatal, "ERROR "+fullPkg+": Marked for removal and needs to be installed.")
			} else {
				params.Logn(LogLevelTrace, fullPkg+": Marked for removal and needs to be installed.")
			}
			installPkgs[fullPkg] = struct{}{}
		} else {
			// if the correct version is already installed not marked for removal we don't want to do anything..
			if params.Mode == ModeReport {
				params.Logn(LogLevelInfo, fullPkg+": Currently installed and not marked for removal.")
			} else {
				params.Logn(LogLevelTrace, fullPkg+": Currently installed and not marked for removal.")
			}
		}
	}

	if len(installPkgs) == 0 && len(uninstallPkgs) == 0 {
		if params.Mode == ModeReport {
			params.Logn(LogLevelInfo, "All required packages are installed.")
		} else {
			params.Logn(LogLevelTrace, "All required packages are installed.")
		}
		return true
	}

	if !pkgsAvailable(installPkgs) {
		params.Logn(LogLevelError, "Not all of the required packages are available in the configured yum repo(s)!")
		return true // TODO return false?
	}

	uninstalled := false
	if len(uninstallPkgs) == 0 {
		uninstalled = true
	}
	params.Logn(LogLevelTrace, "All packages available.. proceeding..")

	if params.Mode == ModeBadass {
		proceed = true
	} else if params.Mode == ModeInteractive && len(uninstallPkgs) > 0 {
		params.Logn(LogLevelInfo, "The following packages must be uninstalled before proceeding:\n  - "+strings.Join(strMapToSlice(uninstallPkgs), "\n  - "))
		proceed = getAnswer("Should I uninstall them now?") && getAnswer("Are you sure you want to proceed with the uninstallation?")
	}

	if proceed && len(uninstallPkgs) > 0 {
		if removePkgs(strMapToSlice(uninstallPkgs)) {
			params.Logn(LogLevelInfo, "Successfully uninstalled the following packages:\n  - "+strings.Join(strMapToSlice(uninstallPkgs), "\n  - "))
			uninstalled = true
		} else {
			params.Logn(LogLevelError, "Unable to uninstall the following packages:\n  - "+strings.Join(strMapToSlice(uninstallPkgs), "\n  - "))
			proceed = false
		}
	}

	if uninstalled && params.Mode == ModeInteractive && len(installPkgs) > 0 {
		params.Logn(LogLevelInfo, "The following packages must be installed:\n  - "+strings.Join(strMapToSlice(installPkgs), "\n  - "))
		proceed = getAnswer("Should I install them now?") && getAnswer("Are you sure you want to proceed with the installation?")
	}

	// TODO this is the pl logic, but is it right? Are packages really installed if uninstall failed?
	if !uninstalled || !proceed || len(installPkgs) == 0 {
		params.Logn(LogLevelInfo, "All of the required packages are installed.")
		return true
	}

	if installPkgs(strMapToSlice(installPkgs)) {
		params.Logn(LogLevelInfo, "Successfully installed the following packages:\n  - "+strings.Join(strMapToSlice(installPkgs), "\n  - "))
		return true
	}

	params.Logn(LogLevelInfo, "Unable to install the following packages:\n  - "+strings.Join(strMapToSlice(installPkgs), "\n  - "))
	return false
}

func removePkgs(pkgs []string) bool {
	actions := []string{"remove", "-y"}
	actions = append(actions, pkgs...)
	return pkgAction(actions)
}

func installPkgs(pkgs []string) bool {
	actions := []string{"install", "-y"}
	actions = append(actions, pkgs...)
	if pkgAction(actions) {
		for _, pkg := range pkgs {
			installTracker(pkg) = struct{}{}
		}
		return true
	}
	return false
}

func getAnswer(params Parmas, question string) bool {
	reader := bufio.NewReader(os.Stdin)
	answer := "notblank"
	for answer != "" && answer != "y" && answer != "n" {
		params.Logn(LogLevelInfo, question+"[Y/n]: ")
		answer = strings.ToLower(strings.TrimSpace(reader.ReadString('\n')))
	}
	return answer != "n"
}

func strMapToSlice(m map[string]struct{}) []string {
	strs := make([]string, 0, len(m))
	for str, _ := range m {
		strs = append(strs, str)
	}
	return strs
}

func pkgsAvailable(installPkgs map[string]struct{}) bool {
	pkgMissing := false
	for pkg, _ := range installPkgs {
		if pkgAction("info", pkg) {
			params.Logn(LogLevelTrace, pkg+" is available")
		} else {
			params.Logn(LogLevelError, pkg+" is not available in the yum repo(s)!")
			pkgMissing := true
		}
	}
	return pkgMissing
}

func pkgAction(actions []string) {
	cmd := exec.Command("/usr/bin/yum", YumOpts, actions...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		params.Logn(LogLevelError, "Execution of "+strings.Join(actions, " ")+" failed! "+err.Error())
		params.Logn(LogLevelError, "Output: "+out)
		return false
	}
	params.Logn(LogLevelTrace, "Successfully executed "+strings.Join(actions, " "))
	// params.Logn(LogLevelDebug, "Output: "+out)
	return true
}

func pkgRequires(name string) []string {
	cmd := exec.Command("/bin/rpm", "-q", "--whatrequires", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		params.Logn(LogLevelError, "pkgRequires failed to get packages: "+err.Error())
		return nil
	}

	return strings.Split(out, "\n")
}

// TODO test
func pkgInstalled(name, version string) []string {
	if version != "" {
		name = name + "-" + version
	}

	cmd := exec.Command("/bin/rpm", "-q", name)
	bytes, err := cmd.CombinedOutput()

	// rpm returns 0 if installed, 1 if not installed
	if err != nil { // TODO test and fix
		return nil
	}

	return strings.Split(out, "\n")
}

func getFullPkgName(name, version string) string {
	return name + "-" + version
}

// type Package struct {
// 	Name    string `json:"name"`
// 	Version string `json:"version"`
// }

func getHeaderComment(toClient *to.Session) string {
	info, err := toClient.SystemInfo()
	if err != nil {
		params.Logn(LogLevelError, "Error getting header comment: "+err.Error())
		params.Logn(LogLevelError, "Did not find tm.toolname!")
		return ""
	}
	toolName, ok := info["tm.toolname"]
	if ok {
		params.Logn(LogLevelInfo, "Found tm.toolname:"+toolName)
	} else {
		params.Logn(LogLevelError, "Did not find tm.toolname!")
	}
	return toolName
}

// getCfgFileList returns the name of this cache's profile, the name of this cache's CDN, and the config file tracker info.
func getCfgFileList(hostnameShort string, toSession *to.Session) (string, string, map[string]ConfigFileInfo, error) {
	url := "/ort/$host_name/ort1"

	ortRef, err := toSession.ORT(hostnameShort)
	if err != nil {
		return "", "", "", nil, err
	}

	profileName := ortRef.Profile.Name
	params.Logn(LogLevelInfo, "Found profile from Traffic Ops: "+profile)

	cdnName := ortRef.Other.CDNName
	params.Logn(LogLevelInfo, "Found CDN_name from Traffic Ops: "+cdnName)

	cfgFiles := map[string]ConfigFileInfo{}
	for cfgFileName, cfgFileLocation := range ortRef.ConfigFiles {
		fnameOnDisk := getFilenameOnDisk(cfgFileName)
		params.Logn(LogLevelInfo, "Found config file (on disk: "+fnameOnDisk+"): "+cfgFileName+" with location: "+cfgFileLocation.Location+")")
		cfgFiles[fnameOnDisk].Location = cfgFileLocation.Location
		cfgFiles[fnameOnDisk].FnameInTrafficOps = cfgFileName
	}

	return profileName, cdnName, cfgFiles
}

// TODO test
func getFilenameOnDisk(configFileName string) {
	prefixToStrip := "to_ext_"
	if len(configFileName) > len(prefixToStrip) && configFileName[:len(prefixToStrip)] == prefixToStrip {
		return configFileName[len(prefixToStrip):]
	}
	return configFileName
}

func deleteOldTmpDirs(params Params) {
	if params.Mode != ModeBadass && params.Mode != ModeInteractive && params.Mode != ModeSyncds {
		return
	}
	smartMkdir(params, TmpBase)
	cleanTmpDirs(params)
}

func smartMkdir(params Params, d string) {
	if _, err := os.Stat(d); err == nil {
		params.Logn(LogLevelTrace, "Directory: "+d+" exists.")
		return
	}

	if params.Mode != ModeBadass && params.Mode != ModeInteractive && params.Mode != ModeSyncds {
		params.Logn(LogLevelError, "Directory: "+d+" doesn't exist, and was not created.")
		return
	}

	if err := os.MkdirAll(statusDir, ModeDir); err != nil {
		params.Logn(LogLevelError, "Creating "+statusDir+" :"+err.Error())
		return
	}

	// TODO move this to the caller; separation of concerns
	if strings.Contains(d, "config_trops") {
		params.Logn(LogLevelDebug, "Temp directory created: "+d+". Config files from Traffic Ops will be placed here for future processing.")
	} else if strings.Contains(d, "config_bkp") {
		params.Logn(LogLevelDebug, "Backup directory created: "+d+". Config files will be backed up here.")
	} else {
		params.Logn(LogLevelDebug, "Directory created: "+d+".")
	}

}

func cleanTmpDirs(params Params) {
	oldTime := time.Now().Sub(time.Day * time.Duration(7))

	files, err := ioutil.ReadDir(TmpBase)
	if err != nil {
		params.Logn(LogLevelError, "Reading temp base '"+TmpBase+"': "+err.Error())
		return
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if f.ModTime().After(oldTime) {
			continue
		}
		fPath := TmpBase + "/" + f.Name()
		params.Logn(LogLevelError, "Deleting directory "+fPath)
		if err := os.Remove(otherStatus); err != nil {
			params.Logn(LogLevelError, "Removing temp dir "+fPath+" :"+err.Error())
		}
	}
}

func checkSyncdsState(params Params, toSession *to.Session, hostnameShort string) UpdateStatusTrafficOps {
	params.Logn(LogLevelDebug, "Checking syncds state.")
	if !doSyncds(params.Mode) {
		return UpdateStatusTrafficOpsNotNeeded
	}

	// The herd is about to get /update/<hostname>

	sleepRand(params, params.Dispersion)

	updRefs, err := toSession.Update(hostnameShort)
	if err != nil {
		params.Log(LogLevelError, err.Error())
		return UpdateStatusTrafficOpsFailed
	}
	updRef := updRefs[0]
	// fmt.Printf("UpdRef: %+v\n", updRef)
	updatePending := updRef.UpdatePending

	if !updatePending && params.Mode == ModeSyncds {
		params.Logn(LogLevelError, "In syncds mode, but no syncds update needs to be applied. I'm outta here.")
		return UpdateStatusTrafficOpsNotNeeded
	}

	if !updatePending {
		params.Logn(LogLevelError, "Traffic Ops is signaling that no update is waiting to be applied.")
	}

	doUpdate := checkSyncdsStateDoUpdate(params, toSession, hostnameShort, updRef)
	if doUpdate {
		return UpdateStatusTrafficOpsNeeded
	} else {
		return UpdateStatusTrafficOpsFailed
	}
}

func doSyncds(mode Mode) bool {
	return mode == ModeSyncds || mode == ModeBadass || mode == ModeReport
}

// checkSyncdsStateDoUpdate returns whether an update should be performed. Errors and information are logged, so when false is returned, callers need not log or do anything.
// TODO change to return error and let caller log.
func checkSyncdsStateDoUpdate(params Params, toSession *to.Session, hostnameShort string, updRef to.Update) bool {
	params.Logn(LogLevelError, "Traffic Ops is signaling that an update is waiting to be applied.")

	if !checkSyncdsStateDoUpdateHandleParentPending(params, toSession, hostnameShort, updRef) {
		return false
	}

	statuses, err = toSession.DataStatus() // &lwp_get("$traffic_ops_host\/datastatus");
	if err != nil {
		params.Logn(LogLevelError, "checkSyncdsStateDoUpdate getting DataStatus: "+err.Error())
		return false
	}

	myStatus := updRef.Status
	params.Logn(LogLevelDebug, "Found "+myStatus+" status from Traffic Ops.")

	statusDir := workingDir + "/stauts"
	statusFile := statusDir + "/" + myStatus

	if _, err := os.Stat(statusFile); os.IsNotExist(err) { // if statusFile doesn't exist
		params.Logn(LogLevelError, "status file "+statusFile+" does not exist.")
		// TODO fail/exit here?
	}

	// remove other status files. E.g. if the current status is REPORTED, remove any ONLINE or OFFLINE files
	// TODO extract method?
	for _, status := range statuses {
		if status.Name == myStatus {
			continue
		}

		otherStatus := statusDir + "/" + status.Name

		if _, err := os.Stat(otherStatus); err == nil { // if the otherStatus file exists
			params.Logn(LogLevelError, "Other status file "+otherStatus+" exists.")
			if params.Mode != ModeReport {
				params.Logn(LogLevelDebug, "Removing "+otherStatus)
				if err := os.Remove(otherStatus); err != nil {
					params.Logn(LogLevelError, "Removing "+otherStatus+" :"+err.Error())
				}
			}
		}
	}

	if params.Mode == ModeReport {
		return true
	}

	// create status directory, if it doesn't exist
	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		if err := os.Mkdir(statusDir, ModeDir); err != nil {
			params.Logn(LogLevelError, "Creating "+statusDir+" :"+err.Error())
			return false
		}
	}

	// create status file, if it doesn't exist
	if _, err := os.Stat(statusFile); os.IsNotExist(err) {
		if f, err := os.Create(statusFile); err != nil {
			params.Logn(LogLevelError, "Creating "+statusFile+" :"+err.Error())
			return false
		} else {
			f.Close()
		}
	}

	return true
}

// checkSyncdsStateDoUpdateHandleParentPending returns whether to continue or bail. Errors will be logged appropriately, so callers only need to do as instructed.
func checkSyncdsStateDoUpdateHandleParentPending(params Params, toSession *to.Session, hostnameShort string, updRef to.Update) bool {
	if !updRef.ParentPending || !params.WaitForParents {
		params.Logn(LogLevelDebug, "Traffic Ops is signaling that my parents do not need an update, or wait_for_parents == 0.")
		return true
	}

	params.Logn(LogLevelError, "Traffic Ops is signaling that my parents need an update.")
	if params.Mode != ModeSyncds {
		return true
	}

	if params.Dispersion > 0 {
		params.Logn(LogLevelWarn, "In syncds mode, sleeping for "+params.Dispersion.String()+" to see if the update my parents need is cleared.")
		for i := 0; i < int(duration.Seconds()); i++ {
			time.Sleep(time.Second)
			params.Log(LogLevelWarn, ".")
		}
		params.Log(LogLevelWarn, "\n")
	}

	var err error
	updRefs, err = toSession.Update(hostnameShort)
	if err != nil {
		params.Log(LogLevelError, "HandleParentPending got Traffic Ops Update: "+err.Error())
		return false
	}
	updRef := updRef[0]

	if updRef.ParentPending {
		params.Logn(LogLevelError, "My parents still need an update, bailing.")
		return false
	}

	params.Logn(LogLevelDebug, "The update on my parents cleared; continuing.")
	return true
}

// TODO move log out of params, so it isn't necessary to pass to everything that needs to log.
func sleepRand(params Params, duration time.Duration) {
	if duration == 0 {
		return
	}
	sleepSeconds := rand.Intn(int(duration.Seconds()))
	params.Log(LogLevelWarn, "WARN: Sleeping for "+fmt.Sprintf("%d", sleepSeconds)+" seconds: ")
	for i := 0; i < sleepSeconds; i++ {
		params.Log(LogLevelWarn, ".")
		time.Sleep(time.Second)
	}
	params.Log(LogLevelWarn, "\n")
}

func run(cmdStr string) (string, error) {
	cmds := strings.Split(cmdStr, " ")
	if len(cmds) < 1 {
		return "", fmt.Errorf("no app")
	}
	cmd := exec.Command(cmds[0], cmds[1:]...)
	bytes, err := cmd.CombinedOutput()
	return string(bytes), err
}

func checkOutput(params Params, out string, err error) {
	if err != nil || strings.Contains(strings.ToLower(out), "error") {
		params.Log(LogLevelError, err.Error())
	}
}

func hostNameShort(hostname string) string {
	dotPos := strings.Index(hostname, ".")
	if dotPos == -1 {
		return hostname
	}
	return hostname[:dotPos]
}

func domainName(hostname string) string {
	dotPos := strings.Index(hostname, ".")
	if dotPos == -1 || len(hostname) < dotPos+2 {
		return ""
	}
	return hostname[dotPos+1:]
}

func SupportedReleases() map[string]struct{} {
	return map[string]struct{}{
		"el6": struct{}{},
		"el7": struct{}{},
	}
}

func SupportedRelease(r string) bool {
	_, ok := SupportedReleases()[strings.ToLower(r)]
	return ok
}

func printUsage() {
	fmt.Printf("Usage:\n")
}

// from https://groups.google.com/forum/#!topic/golang-nuts/Jel8Bb-YwX8
func charsToStr(ca []int8) string {
	s := make([]byte, len(ca))
	var lens int
	for ; lens < len(ca); lens++ {
		if ca[lens] == 0 {
			break
		}
		s[lens] = uint8(ca[lens])
	}
	return string(s[0:lens])
}

func osVersion() (string, error) {
	utsName := unix.Utsname{}
	err := unix.Uname(&utsName)
	if err != nil {
		return "", err
	}
	releaseStr := charsToStr(utsName.Release[0:])
	centosStart := strings.Index(releaseStr, "el")
	if centosStart < 0 {
		return "", fmt.Errorf("unknown release string '%v'", releaseStr)
	}
	elStr := releaseStr[centosStart:]
	if len(elStr) < 3 {
		return "", fmt.Errorf("unknown release string '%v'", releaseStr)
	}
	elN := elStr[0:3]
	return elN, nil
}

// checkOnlyCopyRunning obtains an OS file lock on the app file, returning the locked file. The locked file must not be allowed to go out of scope, else the lock will be released when it is garbage-collected. To release the lock, close the file.
func checkOnlyCopyRunning() (*os.File, error) {
	f, err := os.Open(os.Args[0])
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("error locking file: %v", err)
	}
	return f, nil
}

func checkRunUser(params Params) error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("Error getting user from OS: %v", err)
	}
	if u.Username != "root" && (params.Mode == ModeInteractive || params.Mode == ModeBadass || params.Mode == ModeSyncds) {
		return fmt.Errorf("For interactive, badass, or syncds mode, you must run script as root user. Exiting.\n")
	}
	params.Log(LogLevelTrace, "run user is "+u.Username)
	return nil
}
