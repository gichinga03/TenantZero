export interface SavedQuery {
  name:   string;
  cypher: string;
  group:  string;
}

export const SAVED_QUERIES: SavedQuery[] = [
  {
    group: 'Domain Info',
    name:  'Find All Domain Admins',
    cypher: `MATCH p=(n:Group)<-[:MemberOf*1..]-(m)
WHERE n.name =~ '(?i)domain admins.*'
RETURN p`,
  },
  {
    group: 'Domain Info',
    name:  'Find All Domain Controllers',
    cypher: `MATCH (c:Computer)-[:MemberOf*1..]->(g:Group)
WHERE g.objectid =~ '(?i)S-1-5-.*-516'
RETURN c`,
  },
  {
    group: 'Shortest Paths',
    name:  'Shortest Paths to Domain Admins',
    cypher: `MATCH p=shortestPath((n)-[*1..]->(m:Group))
WHERE m.name =~ '(?i)domain admins.*'
  AND n <> m
RETURN p`,
  },
  {
    group: 'Shortest Paths',
    name:  'Shortest Path from Owned Principals',
    cypher: `MATCH p=shortestPath((n)-[*1..]->(m:Group))
WHERE n.owned = true
  AND m.name =~ '(?i)domain admins.*'
RETURN p`,
  },
  {
    group: 'Dangerous Rights',
    name:  'Find Principals with DCSync Rights',
    cypher: `MATCH p=()-[:DCSync|AllExtendedRights|GenericAll]->(d:Domain)
RETURN p`,
  },
  {
    group: 'Dangerous Rights',
    name:  'Find Computers with Unconstrained Delegation',
    cypher: `MATCH (c:Computer {unconstraineddelegation:true})
WHERE NOT c.name =~ '(?i)DC.*'
RETURN c`,
  },
  {
    group: 'Kerberos',
    name:  'Kerberoastable Users (High Value)',
    cypher: `MATCH (u:User {hasspn:true})
WHERE u.highvalue = true
RETURN u`,
  },
  {
    group: 'Kerberos',
    name:  'AS-REP Roastable Users',
    cypher: `MATCH (u:User {dontreqpreauth:true})
RETURN u`,
  },
  {
    group: 'Local Admins',
    name:  'Find Computers Where Domain Users are Local Admin',
    cypher: `MATCH p=(m:Group)-[:AdminTo]->(c:Computer)
WHERE m.name =~ '(?i)domain users.*'
RETURN p`,
  },
  {
    group: 'Local Admins',
    name:  'Find All Paths from Domain Users to High Value Targets',
    cypher: `MATCH p=shortestPath((n:Group)-[*1..]->(m))
WHERE n.name =~ '(?i)domain users.*'
  AND m.highvalue = true
RETURN p`,
  },
];