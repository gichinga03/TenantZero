export type NodeType =
  | 'User' | 'Computer' | 'Group' | 'Domain'
  | 'GPO'  | 'OU'       | 'Container'
  | 'AZUser' | 'AZGroup' | 'AZTenant' | 'AZApp' | 'AZServicePrincipal';

export interface NodeDef {
  id:                      string;
  label:                   string;
  type:                    NodeType;
  domain:                  string;
  sid?:                    string;
  guid?:                   string;
  adminCount?:             boolean;
  hasspn?:                 boolean;
  enabled?:                boolean;
  owned?:                  boolean;
  highvalue?:              boolean;
  os?:                     string;
  pwdlastset?:             string;
  lastlogon?:              string;
  description?:            string;
  unconstraineddelegation?:boolean;
  functionallevel?:        string;
  members?:                number;
  pwdneverexpires?:        boolean;
  dontreqpreauth?:         boolean;
  sensitive?:              boolean;
}

export interface EdgeDef {
  source: string;
  target: string;
  label:  string;
}

export interface AclEntry {
  principal:   string;
  right:       string;
  inherited:   boolean;
  grantedBy?:  string;
}