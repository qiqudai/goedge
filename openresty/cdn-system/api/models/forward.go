package models

import (
  "encoding/json"
  "time"

  "gorm.io/gorm"
)

type ForwardOrigin struct {
  Address string `json:"address"`
  Weight  int    `json:"weight"`
  Enable  bool   `json:"enable"`
}

// Forward corresponds to `forward` table.
type Forward struct {
  ID            int64  `json:"id" gorm:"primaryKey"`
  UserID        int64  `json:"uid" gorm:"column:uid;index"`
  UserPackageID int64  `json:"user_package_id" gorm:"column:user_package"`
  NodeGroupID   int64  `json:"node_group_id" gorm:"column:node_group_id"`
  Enable        bool   `json:"enable"`
  State         string `json:"state"`
  Remark        string `json:"remark" gorm:"column:remark;type:text"`
  Cname         string `json:"cname" gorm:"column:cname;type:varchar(255)"`

  ListenPortsRaw string          `json:"-" gorm:"column:listen_ports;type:text"`
  ListenPorts    []string        `json:"listen_ports" gorm:"-"`
  OriginsRaw     string          `json:"-" gorm:"column:origins;type:text"`
  Origins        []ForwardOrigin `json:"origins" gorm:"-"`
  SettingsRaw    string          `json:"-" gorm:"column:settings;type:longtext"`
  Settings       map[string]interface{} `json:"settings" gorm:"-"`

  CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
  UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (Forward) TableName() string {
  return "forward"
}

func (f *Forward) BeforeSave(tx *gorm.DB) (err error) {
  if len(f.ListenPorts) > 0 {
    b, _ := json.Marshal(f.ListenPorts)
    f.ListenPortsRaw = string(b)
  }
  if len(f.Origins) > 0 {
    b, _ := json.Marshal(f.Origins)
    f.OriginsRaw = string(b)
  }
  if f.Settings != nil {
    b, _ := json.Marshal(f.Settings)
    f.SettingsRaw = string(b)
  }
  return
}

func (f *Forward) AfterFind(tx *gorm.DB) (err error) {
  if f.ListenPortsRaw != "" {
    _ = json.Unmarshal([]byte(f.ListenPortsRaw), &f.ListenPorts)
  }
  if f.OriginsRaw != "" {
    _ = json.Unmarshal([]byte(f.OriginsRaw), &f.Origins)
  }
  if f.SettingsRaw != "" {
    _ = json.Unmarshal([]byte(f.SettingsRaw), &f.Settings)
  }
  return
}

type ForwardGroup struct {
  ID        int64     `json:"id" gorm:"primaryKey"`
  Name      string    `json:"name" gorm:"column:name;type:varchar(64)"`
  Remark    string    `json:"remark" gorm:"column:remark;type:varchar(255)"`
  CreatedAt time.Time `json:"create_at" gorm:"column:create_at"`
  UpdatedAt time.Time `json:"update_at" gorm:"column:update_at"`
}

func (ForwardGroup) TableName() string {
  return "forward_group"
}

type ForwardGroupRelation struct {
  ID        int64 `json:"id" gorm:"primaryKey"`
  ForwardID int64 `json:"forward_id" gorm:"column:forward_id;index"`
  GroupID   int64 `json:"group_id" gorm:"column:group_id;index"`
}

func (ForwardGroupRelation) TableName() string {
  return "forward_group_relation"
}
