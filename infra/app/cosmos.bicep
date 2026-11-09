param name string
param location string = resourceGroup().location
param tags object = {}
param principalId string
param userPrincipalId string = ''

resource account 'Microsoft.DocumentDB/databaseAccounts@2024-05-15' = {
  name: name
  location: location
  tags: tags
  kind: 'GlobalDocumentDB'
  properties: {
    databaseAccountOfferType: 'Standard'
    disableLocalAuth: true
    publicNetworkAccess: 'Enabled'
    minimalTlsVersion: 'Tls12'
    consistencyPolicy: {
      defaultConsistencyLevel: 'Session'
    }
    capabilities: [{ name: 'EnableServerless' }]
    locations: [{
      locationName: location
      failoverPriority: 0
      isZoneRedundant: false
    }]
  }
}

resource database 'Microsoft.DocumentDB/databaseAccounts/sqlDatabases@2024-05-15' = {
  parent: account
  name: 'ToDoList'
  properties: {
    resource: { id: 'ToDoList' }
  }
}

resource items 'Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers@2024-05-15' = {
  parent: database
  name: 'Items'
  properties: {
    resource: {
      id: 'Items'
      partitionKey: {
        paths: ['/category']
        kind: 'Hash'
        version: 2
      }
    }
  }
}

resource leases 'Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers@2024-05-15' = {
  parent: database
  name: 'leases'
  properties: {
    resource: {
      id: 'leases'
      partitionKey: {
        paths: ['/id']
        kind: 'Hash'
        version: 2
      }
    }
  }
}

var contributorRoleId = '${account.id}/sqlRoleDefinitions/00000000-0000-0000-0000-000000000002'

resource appRole 'Microsoft.DocumentDB/databaseAccounts/sqlRoleAssignments@2024-05-15' = {
  parent: account
  name: guid(account.id, principalId, database.name)
  properties: {
    principalId: principalId
    roleDefinitionId: contributorRoleId
    scope: '${account.id}/dbs/${database.name}'
  }
}

resource writerRole 'Microsoft.DocumentDB/databaseAccounts/sqlRoleAssignments@2024-05-15' = if (!empty(userPrincipalId)) {
  parent: account
  name: guid(account.id, userPrincipalId, items.name)
  properties: {
    principalId: userPrincipalId
    roleDefinitionId: contributorRoleId
    scope: '${account.id}/dbs/${database.name}/colls/${items.name}'
  }
}

output endpoint string = account.properties.documentEndpoint
