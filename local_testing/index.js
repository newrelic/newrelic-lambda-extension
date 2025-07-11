'use strict'

process.env.NEW_RELIC_APP_NAME = process.env.NEW_RELIC_APP_NAME || process.env.AWS_LAMBDA_FUNCTION_NAME
process.env.NEW_RELIC_DISTRIBUTED_TRACING_ENABLED = process.env.NEW_RELIC_DISTRIBUTED_TRACING_ENABLED || 'true'
process.env.NEW_RELIC_NO_CONFIG_FILE = process.env.NEW_RELIC_NO_CONFIG_FILE || 'true'
process.env.NEW_RELIC_TRUSTED_ACCOUNT_KEY =
  process.env.NEW_RELIC_TRUSTED_ACCOUNT_KEY || process.env.NEW_RELIC_ACCOUNT_ID

if (process.env.LAMBDA_TASK_ROOT && typeof process.env.NEW_RELIC_SERVERLESS_MODE_ENABLED !== 'undefined') {
  delete process.env.NEW_RELIC_SERVERLESS_MODE_ENABLED
}

const newrelic = require('newrelic')
const fs = require('node:fs')
const path = require('node:path')

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

  const handlerToWrap = parts[parts.length - 1]
  const moduleToImport = handler.slice(0, handler.lastIndexOf('.'))
  return { moduleToImport, handlerToWrap }
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

function getModuleWithRequire(appRoot, moduleToImport) {
  const modulePath = path.resolve(appRoot, moduleToImport)
  const validExtensions = ['.cjs', '.js']
  const fullModulePath = getFullyQualifiedModulePath(modulePath, validExtensions)

  try {
    return require(fullModulePath)
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

let wrappedHandler
let patchedHandlerPromise

const { LAMBDA_TASK_ROOT = '.' } = process.env
const { moduleToImport, handlerToWrap } = getHandlerPath()

if (process.env.NEW_RELIC_USE_ESM === 'true') {
  patchedHandlerPromise = getHandler().then(userHandler => {
    return newrelic.setLambdaHandler(userHandler)
  })
} else {
  wrappedHandler = newrelic.setLambdaHandler(getHandlerSync())
}

async function getHandler() {
  const userHandler = (await getModuleWithImport(LAMBDA_TASK_ROOT, moduleToImport))[handlerToWrap]
  validateHandlerDefinition(userHandler, handlerToWrap, moduleToImport)

  return userHandler
}

function getHandlerSync() {
  const userHandler = getModuleWithRequire(LAMBDA_TASK_ROOT, moduleToImport)[handlerToWrap]
  validateHandlerDefinition(userHandler, handlerToWrap, moduleToImport)

  return userHandler
}

async function patchHandler() {
  const args = Array.prototype.slice.call(arguments)
  return patchedHandlerPromise
    .then(_wrappedHandler => _wrappedHandler.apply(this, args))
}

let handler 

if (process.env.NEW_RELIC_USE_ESM === 'true') {
  handler = patchHandler
} else {
  handler = wrappedHandler
}


module.exports = {
  handler,
  getHandlerPath
}
