import newrelic from 'newrelic'
import fs from 'node:fs'
import path from 'node:path'

process.env.NEW_RELIC_APP_NAME = process.env.NEW_RELIC_APP_NAME || process.env.AWS_LAMBDA_FUNCTION_NAME
process.env.NEW_RELIC_DISTRIBUTED_TRACING_ENABLED = process.env.NEW_RELIC_DISTRIBUTED_TRACING_ENABLED || 'true'
process.env.NEW_RELIC_NO_CONFIG_FILE = process.env.NEW_RELIC_NO_CONFIG_FILE || 'true'
process.env.NEW_RELIC_TRUSTED_ACCOUNT_KEY =
  process.env.NEW_RELIC_TRUSTED_ACCOUNT_KEY || process.env.NEW_RELIC_ACCOUNT_ID

if (process.env.LAMBDA_TASK_ROOT && typeof process.env.NEW_RELIC_SERVERLESS_MODE_ENABLED !== 'undefined') {
  delete process.env.NEW_RELIC_SERVERLESS_MODE_ENABLED
}
function getNestedHandler(object, nestedProperty) {
  return nestedProperty.split('.').reduce((nested, key) => {
    return nested && nested[key]
  }, object)
}
function getHandlerPath() {
  let handler
  const { NEW_RELIC_LAMBDA_HANDLER } = process.env

  if (!NEW_RELIC_LAMBDA_HANDLER) {
    throw new Error('No NEW_RELIC_LAMBDA_HANDLER environment variable set.')
  } else {
    handler = NEW_RELIC_LAMBDA_HANDLER
  }

  const parts = handler.split('.')

  if (parts.length < 2) {
    throw new Error(
      `Improperly formatted handler environment variable: ${handler}`
    )
  }

  const lastSlashIndex = handler.lastIndexOf('/') + 1
  const firstDotAfterSlash = handler.indexOf('.', lastSlashIndex)
  const moduleToImport = handler.slice(0, firstDotAfterSlash)
  const handlerToWrap = handler.slice(firstDotAfterSlash + 1)

  return {moduleToImport, handlerToWrap}
}

function handleRequireImportError(e, moduleToImport) {
  if (e.code === 'MODULE_NOT_FOUND') {
    return new Error(`Unable to import module '${moduleToImport}'`)
  }
  return e
}

function getFullyQualifiedModulePath(modulePath, extensions) {
  let fullModulePath

  extensions.forEach((extension) => {
    const filePath = modulePath + extension
    if (fs.existsSync(filePath)) {
      fullModulePath = filePath
      return
    }
  })

  if (!fullModulePath) {
    throw new Error(
      `Unable to resolve module file at ${modulePath} with the following extensions: ${extensions.join(',')}`
    )
  }

  return fullModulePath
}

async function getModuleWithImport(appRoot, moduleToImport) {
  const modulePath = path.resolve(appRoot, moduleToImport)
  const validExtensions = ['.mjs', '.js']
  const fullModulePath = getFullyQualifiedModulePath(modulePath, validExtensions)

  try {
    return await import(fullModulePath)
  } catch (err) {
    throw handleRequireImportError(err, moduleToImport)
  }
}

function validateHandlerDefinition(userHandler, handlerName, moduleName) {
  if (typeof userHandler === 'undefined') {
    throw new Error(
      `Handler '${handlerName}' missing on module '${moduleName}'`
    )
  }

  if (typeof userHandler !== 'function') {
    throw new Error(
      `Handler '${handlerName}' from '${moduleName}' is not a function`
    )
  }
}

const { LAMBDA_TASK_ROOT = '.' } = process.env
const { moduleToImport, handlerToWrap } = getHandlerPath()

const userHandler  = await getHandler() 
const handler = newrelic.setLambdaHandler(userHandler)

async function getHandler() {
  const userModule = await getModuleWithImport(LAMBDA_TASK_ROOT, moduleToImport)
  const userHandler = getNestedHandler(userModule, handlerToWrap)
  validateHandlerDefinition(userHandler, handlerToWrap, moduleToImport)

  return userHandler
}
  
export { handler, getHandlerPath }

