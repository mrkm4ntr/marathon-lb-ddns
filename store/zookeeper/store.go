package zookeeper

import (
	"github.com/samuel/go-zookeeper/zk"
	"log"
	"strings"
	"time"
)

type zkStore struct {
	group     string
	conn      *zk.Conn
	ipNodeVer int32
}

func (zs *zkStore) GetIPAddresses() ([]string, error) {
	data, stat, err := zs.conn.Get("/" + zs.group + "/IP")
	if err != nil {
		return nil, err
	}
	zs.ipNodeVer = stat.Version
	ipAddresses := strings.Split(string(data), ",")
	return ipAddresses, nil
}

func (zs *zkStore) SetIPAddresses(ipAddresses []string) error {
	data := []byte(strings.Join(ipAddresses, ","))
	stat, err := zs.conn.Set("/"+zs.group+"/IP", data, zs.ipNodeVer)
	zs.ipNodeVer = stat.Version
	return err
}

func (zs *zkStore) ListCNames() ([]string, error) {
	cNames, _, err := zs.conn.Children("/" + zs.group + "/CNAME")
	if err != nil {
		return nil, err
	}
	return cNames, nil
}

func (zs *zkStore) AddCName(cName string) error {
	_, err := zs.conn.Create("/"+zs.group+"/CNAME/"+cName, []byte{}, 0, zk.WorldACL(zk.PermAll))
	return err
}

func (zs *zkStore) RemoveCName(cName string) error {
	return zs.conn.Delete("/"+zs.group+"/CNAME/"+cName, 0)
}

func (zs *zkStore) Connect(url string) error {
	conn, _, err := zk.Connect([]string{url}, time.Second)
	if err != nil {
		return err
	}
	zs.conn = conn
	return nil
}

func (zs *zkStore) Bootstrap(group string) {
	zs.group = group
	_, err := zs.conn.Create("/"+group, []byte{}, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Println(err)
	}
	_, err = zs.conn.Create("/"+group+"/CNAME", []byte{}, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Println(err)
	}
	_, err = zs.conn.Create("/"+group+"/IP", []byte(""), 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Println(err)
	}
}

var instance = &zkStore{}

func GetInstance() *zkStore {
	return instance
}
