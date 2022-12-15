package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"text/template"
)

//идея добавления логирования запросов и с каких устройств и также игнорирование запросов со неизвестных источников

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Чтобы получить конфинг коммутатора - необходимо добавить к URL: /BuildConfig?id=\"ID устройства\"")
}

func GetSpecialPortsInfo(Id int) ([]Ethernet, error) {
	strId := strconv.Itoa(Id)
	interfaceInfo, err := readJsonMapHttp("http://confdata2.proxicom.ru/reqs/" + strId + ".json")
	if err != nil {
		return nil, err
	}
	specialPortList, ok := interfaceInfo["SpecialPorts"].(map[string]any)

	portsInfo := make([]Ethernet, len(specialPortList))
	if !ok {
		err := errors.New("error while getting \"SpecialPorts\" in ./switches/" + strId + ".json")
		return nil, err
	}
	j := 0
	for i, element := range specialPortList {
		elMap, ok := element.(map[string]any)
		if !ok {
			err := errors.New("error while getting \"" + i + "\" in ./switches/" + strId + ".json")
			return nil, err
		}
		portsInfo[j].EthName = i
		portsInfo[j].PortRole, ok = elMap["PortRole"].(string)
		if !ok {
			err := errors.New("error while getting \"" + i + "\" \"PortRole\" in ./switches/" + strId + ".json")
			return nil, err
		}
		switch portsInfo[j].PortRole {
		case "QinQCustomer":
			portsInfo[j].Vlan = "861"
			portsInfo[j].Vlan, ok = elMap["AccessVlan"].(string)
			if !ok {
				err := errors.New("error while getting \"" + i + "\" \"AccessVlan\" in ./switches/" + strId + ".json")
				return nil, err
			}
		case "AloneHole":
			portsInfo[j].Vlan = "-1"
		case "uplink":
			trunkVlans, ok := elMap["TrunkVlans"].([]any)
			if !ok {
				err := errors.New("error while getting \"" + i + "\" \"TrunkVlans\" in ./switches/" + strId + ".json")
				return nil, err
			}
			portsInfo[j].Vlan = "-1"
			j2 := 0
			portsInfo[j].TrunkVlans = make([]string, len(trunkVlans))
			for _, trunkElement := range trunkVlans {
				portsInfo[j].TrunkVlans[j2], ok = trunkElement.(string)
				if !ok {
					err := errors.New("error while getting \"" + i + "\" \"TrunkVlans element\" in ./switches/" + strId + ".json")
					return nil, err
				}
				j2++
			}
		}
		j++
	}
	return portsInfo, nil
}
func GetSwitchInfo(Id int) (Switch, error) {
	strId := strconv.Itoa(Id)
	var sw Switch
	//Раскрытие глобальной инфы
	globalInfo, err := readJsonMapHttp("http://confdata2.proxicom.ru/global.json")
	//globalInfo, err := readJsonMapFile("./global.json")
	//Раскрытие файла инфы интерфейсов N-го коммутатора

	//Получение информации о настройках коммутатора
	switchList, ok := globalInfo["SwitchList"].(map[string]any)
	if !ok {
		err := errors.New("error while getting \"SwitchList\" in global.json")
		return sw, err
	}
	switchInfo, ok := switchList["SwId-"+strId].(map[string]any)
	if !ok {
		err := errors.New("error while getting \"SwitchList\" data in global.json")
		return sw, err
	}
	switchModels, ok := globalInfo["models"].(map[string]any)
	if !ok {
		err := errors.New("error while getting \"models\" data in global.json")
		return sw, err
	}
	switchName, ok := switchInfo["SwitchModel"].(string)
	if !ok {
		err := errors.New("error while getting SwId-" + strId + "'s \"SwitchModel\" data in global.json")
		return sw, err
	}
	switchPortsInfo, ok := switchModels[switchName].(map[string]any)
	if !ok {
		err := errors.New("error while getting \"" + switchName + "\" data in global.json")
		return sw, err
	}
	switchPortList, ok := switchPortsInfo["PortList"].([]any)
	if !ok {
		err := errors.New("error while getting \"" + switchName + "\"'s \"PortList\" data in global.json")
		return sw, err
	}
	defCustomerVlan, ok := switchInfo["DefaultCustomerVlan"].(string)
	if !ok {
		err := errors.New("error while getting SwId-" + strId + "'s \"DefaultCustomerVlan\" data in global.json")
		return sw, err
	}
	//Заполнение структуры

	sw.Hostname = strId
	sw.Ifaces = make([]Ethernet, len(switchPortList))

	specialPortsInfo, err := GetSpecialPortsInfo(Id)
	if err != nil {
		log.Println(err, " -> line 127")
		return sw, err
	}

	vlansForAlone, ok := globalInfo["VlansForAlone"].([]any)
	if !ok {
		err := errors.New("error while getting \"VlansForAlone\" data in global.json")
		return sw, err
	}

	if len(vlansForAlone) != 2 {
		err := errors.New("error \"VlansForAlone\" has incorrect format in global.json")
		return sw, err
	}
	minAloneVlan := -1
	maxAloneVlan := -1
	//Установка диопазона AloneVlans
	{
		minAloneVlanStr, ok := vlansForAlone[0].(string)
		if !ok {
			err := errors.New("error while getting \"VlansForAlone\" min value in global.json")
			return sw, err
		}
		maxAloneVlanStr, ok := vlansForAlone[1].(string)
		if !ok {
			err := errors.New("error while getting \"VlansForAlone\" max value in global.json")
			return sw, err
		}

		minAloneVlan, err = strconv.Atoi(minAloneVlanStr)
		if err != nil {
			err := errors.New("error \"VlansForAlone\" min value has incorrect format in global.json")
			return sw, err
		}
		maxAloneVlan, err = strconv.Atoi(maxAloneVlanStr)
		if err != nil {
			err := errors.New("error \"VlansForAlone\" max value has incorrect format in global.json")
			return sw, err
		}
	}
	curAloneVlan := minAloneVlan

	//Загрузка SpecialPorts
	//Выявление из портов особенного и задание им соответсвующих значений
	//Остальным назначить стандарт
	for i := 0; i < len(sw.Ifaces); i++ {
		ethName, ok := switchPortList[i].(string)
		if !ok {
			err := errors.New("error while getting \"" + switchName + "\"'s \"PortList\" Ethernet name in global.json")
			return sw, err
		}
		sw.Ifaces[i] =
			Ethernet{ethName, defCustomerVlan, "Customer", nil}

		for _, x := range specialPortsInfo {
			if x.EthName == ethName {
				sw.Ifaces[i].PortRole = x.PortRole
				if sw.Ifaces[i].PortRole == "AloneHole" || sw.Ifaces[i].PortRole == "uplink" {
					sw.Ifaces[i].Vlan = strconv.Itoa(curAloneVlan)
					curAloneVlan++
					if curAloneVlan > maxAloneVlan {
						err := errors.New("error VlansForAlone out of range (in global.json)")
						return sw, err
					}
				} else {
					sw.Ifaces[i].Vlan = x.Vlan
				}
				sw.Ifaces[i].TrunkVlans = x.TrunkVlans
			}
		}
	}

	//Загрузка ControlVlans
	ControlVlans, ok := switchInfo["ControlVlans"].(map[string]any)
	if !ok {
		err := errors.New("error while getting SwId-" + strId + "'s \"ControlVlans\" data in global.json")
		return sw, err
	}
	sw.ControlVlans = make([]Vlan, len(ControlVlans))
	{
		j := 0
		for i, element := range ControlVlans {
			elMap, ok := element.(map[string]any)
			if !ok {
				err := errors.New("error while getting SwId-" + strId + "'s \"ControlVlans\" " + i + "'s in global.json")
				return sw, err
			}
			ip, ok := elMap["ip"].(string)
			if !ok {
				err := errors.New("error while getting SwId-" + strId + "'s \"ControlVlans\" ip number data in global.json")
				return sw, err
			}
			sw.ControlVlans[j].VlanId = i
			sw.ControlVlans[j].IP = ip
			j++
		}
	}
	//sw.ControlVlans получается неупорядоченным. Поэтому сортируется
	sort.Slice(sw.ControlVlans, func(i, j int) bool {
		first, err := strconv.Atoi(sw.ControlVlans[i].VlanId)
		second, err := strconv.Atoi(sw.ControlVlans[j].VlanId)
		if err != nil {
			log.Println(err, " -> line 229")
			return false
		}
		return first < second
	})
	sw.Gateway, ok = switchInfo["IpDefaultGateway"].(string)
	if !ok {
		err := errors.New("error while getting SwId-" + strId + "'s \"IpDefaultGateway\" data in global.json")
		return sw, err
	}
	return sw, nil
}

func buildConfig(w http.ResponseWriter, r *http.Request) {
	strId := r.FormValue("id")

	//	Генерация конфига

	//1
	id, err := strconv.Atoi(strId)
	if err != nil {
		log.Println(err, "-> line 251")
		fmt.Fprintln(w, "+++\n", err, " -> line 251")
		return
	}
	sw, err := GetSwitchInfo(id)
	if err != nil {
		log.Println(err, " -> line 257")
		fmt.Fprintln(w, "+++\n", err, " -> line 257")
		return
	}
	//Взятие соотвествующего шаблона из имеющегося списка

	tmplPath := "./tmpls/common.txt"
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		log.Println(err, "-> line 265")
		fmt.Fprintln(w, "+++\n", err, "-> line 265")
		return
	}
	//Вставка структуры и вывод полученного
	if err := tmpl.Execute(w, sw); err != nil {
		log.Println(err, " -> line 271")
		fmt.Fprintln(w, "+++\n", err, " -> line 271")
		return
	}
}

func main() {
	//Задание начальных настроек
	var port int

	flag.IntVar(&port, "port", 8080, "Server listen port")
	flag.Parse()
	fmt.Println(":" + strconv.Itoa(port))

	//Запуск сервера
	http.HandleFunc("/BuildConfig", buildConfig)
	http.HandleFunc("/", index)

	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		log.Print(err)
		fmt.Println(" -> line 293")
	}
}
