import type { EdgeDef } from '../types';

export const EDGES: EdgeDef[] = [
  { source: 'jsmith',     target: 'WKSTN01',    label: 'HasSession'         },
  { source: 'jsmith',     target: 'IT_Support',  label: 'MemberOf'           },
  { source: 'awalker',    target: 'IT_Support',  label: 'MemberOf'           },
  { source: 'awalker',    target: 'meridian',    label: 'WriteDACL'          },
  { source: 'awalker',    target: 'DA',          label: 'AddMember'          },
  { source: 'svc_backup', target: 'DA',          label: 'MemberOf'           },
  { source: 'bthompson',  target: 'RDP_Users',   label: 'MemberOf'           },
  { source: 'mreyes',     target: 'RDP_Users',   label: 'MemberOf'           },
  { source: 'IT_Support', target: 'WKSTN01',     label: 'AdminTo'            },
  { source: 'IT_Support', target: 'WKSTN02',     label: 'AdminTo'            },
  { source: 'DA',         target: 'DC01',        label: 'AdminTo'            },
  { source: 'DA',         target: 'SRV01',       label: 'AdminTo'            },
  { source: 'DA',         target: 'meridian',    label: 'GetChangesAll'      },
  { source: 'EA',         target: 'meridian',    label: 'GenericAll'         },
  { source: 'RDP_Users',  target: 'WKSTN01',     label: 'CanRDP'             },
  { source: 'RDP_Users',  target: 'WKSTN02',     label: 'CanRDP'             },
  { source: 'krbtgt',     target: 'meridian',    label: 'MemberOf'           },
  { source: 'DC01',       target: 'meridian',    label: 'TrustedBy'          },
  { source: 'awalker',    target: 'svc_backup',  label: 'ForceChangePassword'},
];

export const ATTACK_PATH = ['jsmith', 'IT_Support', 'WKSTN01', 'DA', 'DC01'];

export const ACLS: Record<string, Array<{ principal: string; right: string; inherited: boolean }>> = {
  meridian: [
    { principal: 'DA',       right: 'GenericAll',      inherited: false },
    { principal: 'EA',       right: 'GenericAll',      inherited: false },
    { principal: 'awalker',  right: 'WriteDACL',       inherited: false },
    { principal: 'DA',       right: 'GetChangesAll',   inherited: false },
  ],
  DC01: [
    { principal: 'DA',       right: 'GenericAll',      inherited: true  },
    { principal: 'EA',       right: 'GenericAll',      inherited: true  },
  ],
  DA: [
    { principal: 'awalker',  right: 'AddMember',       inherited: false },
    { principal: 'EA',       right: 'GenericAll',      inherited: false },
  ],
};