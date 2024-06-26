/*
 * Copyright 2024 Gabriel Cataldo
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GabrielHCataldo/go-errors/errors"
	"github.com/GabrielHCataldo/go-helper/helper"
	"github.com/GabrielHCataldo/go-logger/logger"
	"github.com/GabrielHCataldo/gopen-gateway/internal/app"
	"github.com/GabrielHCataldo/gopen-gateway/internal/app/controller"
	"github.com/GabrielHCataldo/gopen-gateway/internal/app/mapper"
	"github.com/GabrielHCataldo/gopen-gateway/internal/app/model/dto"
	"github.com/GabrielHCataldo/gopen-gateway/internal/domain/model/vo"
	"github.com/GabrielHCataldo/gopen-gateway/internal/domain/service"
	"github.com/GabrielHCataldo/gopen-gateway/internal/infra"
	"github.com/GabrielHCataldo/gopen-gateway/internal/infra/middleware"
	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/xeipuuv/gojsonschema"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"
)

// gopenJsonResult is a constant string representing the filepath of the Gopen JSON result file.
const gopenJsonResult = "./gopen.json"

// gopenJsonSchema is a constant string representing the URI of the JSON schema file.
const gopenJsonSchema = "file://./json-schema.json"

// loggerOptions is the configuration options for the logger package.
// It specifies the custom text to be displayed after the log prefix.
var loggerOptions = logger.Options{
	CustomAfterPrefixText: "CMD",
}

// gopenApp is an instance of the app.Gopen interface that represents the functionality of a Gopen server.
// It is used to start and shutdown the Gopen application by invoking its ListerAndServer() and
// Shutdown(ctx context.Context) error methods.
var gopenApp app.Gopen

// main is the entry point of the application.
// It starts the application by performing the following steps:
// 1. Prints a starting message using printInfoLog.
// 2. Reads the environment argument from command line.
// 3. Loads the default environment variables for Gopen using loadGopenDefaultEnvs.
// 4. Loads the environment variables indicated by the env argument using loadGopenEnvs.
// 5. Builds the Gopen configuration DTO by calling loadGopenJson.
// 6. Starts the application by calling startApp in a separate goroutine.
// 7. Waits for interrupt signal to stop the application.
// 8. Removes the Gopen JSON result file using removeGopenJsonResult.
// 9. Prints a message indicating the application has stopped using logger.
func main() {
	printInfoLog("Starting..")

	// inicializamos o valor env para obter como argumento de aplicação
	var env string
	if helper.IsLessThanOrEqual(os.Args, 1) {
		panic(errors.New("Please enter ENV as second argument! ex: dev, prd"))
	}
	env = os.Args[1]

	// carregamos as variáveis de ambiente padrão da app
	loadGopenDefaultEnvs()

	// carregamos as variáveis de ambiente indicada
	loadGopenEnvs(env)

	// construímos o dto de configuração do Gopen
	gopenDTO := loadGopenJson(env)

	// inicializamos a aplicação
	go startApp(env, gopenDTO)

	// seguramos a goroutine principal esperando que aplicação seja interrompida
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		// removemos o arquivo json que foi usado
		removeGopenJsonResult()
		// imprimimos que a aplicação foi interrompida
		logger.Info("Gopen Stopped!")
	}
}

// loadGopenDefaultEnvs loads the default environment variables for Gopen.
func loadGopenDefaultEnvs() {
	// carregamos as envs padrões do Gopen
	printInfoLog("Loading Gopen envs default...")
	if err := godotenv.Load("./internal/infra/config/.env"); helper.IsNotNil(err) {
		panic(errors.New("Error load Gopen envs default:", err))
	}
}

// loadGopenEnvs loads the environment variables indicated by the argument 'env'.
// It retrieves the file URI based on the 'env' and loads the environment variables from that file.
// It logs the process and prints a warning if there is an error loading the environment variables.
func loadGopenEnvs(env string) {
	// carregamos as envs indicada no arg
	fileEnvUri := getFileEnvUri(env)
	printInfoLogf("Loading Gopen envs from uri: %s...", fileEnvUri)
	if err := godotenv.Load(fileEnvUri); helper.IsNotNil(err) {
		printWarningLog("Error load Gopen envs from uri:", fileEnvUri, "err:", err)
	}
}

// loadGopenJson loads the Gopen configuration from a JSON file and returns it as a dto.Gopen object.
// The function takes an environment string as a parameter, which is used to determine the file path.
// The file path is generated by calling the getFileJsonUri function with the environment string.
// The function reads the file contents using the os.ReadFile function and stores them in the fileJsonBytes variable.
// Then, it calls the fillEnvValues function to replace environment variable placeholders in the JSON with their actual values.
// After that, it validates the JSON schema by calling the validateJsonBySchema function.
// If the validation fails, the function panics with the error message.
// Next, it converts the JSON bytes into a dto.Gopen object by using the helper.ConvertToDest function.
// If the conversion fails, the function panics with the error message.
// Finally, the function returns the dto.Gopen object.
func loadGopenJson(env string) *dto.Gopen {
	// carregamos o arquivo de json de configuração do Gopen
	fileJsonUri := getFileJsonUri(env)
	printInfoLogf("Loading Gopen json from file: %s...", fileJsonUri)
	fileJsonBytes, err := os.ReadFile(fileJsonUri)
	if helper.IsNotNil(err) {
		panic(errors.New("Error read martini config from file json:", fileJsonUri, "err:", err))
	}

	// preenchemos os valores de variável de ambiente com a sintaxe pre-definida
	fileJsonBytes = fillEnvValues(fileJsonBytes)

	// validamos o schema
	if err = validateJsonBySchema(fileJsonUri, fileJsonBytes); helper.IsNotNil(err) {
		panic(err)
	}

	// convertemos o valor em bytes em DTO
	var gopenDTO dto.Gopen
	err = helper.ConvertToDest(fileJsonBytes, &gopenDTO)
	if helper.IsNotNil(err) {
		panic(errors.New("Error parse Gopen json file to DTO:", err))
	}

	// retornamos o DTO que é a configuração do Gopen
	return &gopenDTO
}

// fillEnvValues fills the environment variables in a JSON string using the $word syntax.
// The function takes a byte array of the JSON string as input and returns the modified JSON string as a byte array.
// It searches for all occurrences of $word in the JSON string and replaces them with the corresponding environment variable value.
// The function uses regular expressions to find all $word occurrences and os.Getenv() to get the environment variable value.
// If a valid value is found, it replaces the $word with the value in the JSON string.
// The function prints the number of environment variable values found and successfully filled during the process.
// It also uses the helper functions 'SimpleConvertToString' and 'SimpleConvertToBytes' to convert the byte array to string and vice versa.
func fillEnvValues(gopenBytesJson []byte) []byte {
	// todo: aceitar campos não string receber variável de ambiente também
	//  foi pensado que talvez utilizar campos string e any para isso, convertendo para o tipo desejado apenas
	//  quando objeto de valor for montado

	printInfoLog("Filling environment variables with $word syntax..")

	// convertemos os bytes do gopen json em string
	gopenStrJson := helper.SimpleConvertToString(gopenBytesJson)

	// compilamos o regex indicando um valor de env $API_KEY por exemplo
	regex := regexp.MustCompile(`\$\w+`)
	// damos o find pelo regex
	words := regex.FindAllString(gopenStrJson, -1)

	// imprimimos todas as palavras encontradas a ser preenchidas
	printInfoLog(len(words), "environment variable values were found to fill in!")

	// inicializamos o contador de valores processados
	count := 0
	for _, word := range words {
		// replace do valor padrão $
		envKey := strings.ReplaceAll(word, "$", "")
		// obtemos o valor da env pela chave indicada
		envValue := os.Getenv(envKey)
		// caso valor encontrado, damos o replace da palavra encontrada pelo valor
		if helper.IsNotEmpty(envValue) {
			gopenStrJson = strings.ReplaceAll(gopenStrJson, word, envValue)
			count++
		}
	}
	// imprimimos a quantidade de envs preenchidas
	printInfoLog(count, "environment variables successfully filled!")

	// convertemos esse novo
	return helper.SimpleConvertToBytes(gopenStrJson)
}

// validateJsonBySchema validates a JSON file against a given schema.
// It takes the file JSON URI and the file JSON bytes as inputs.
// The function starts by printing a log message indicating that it is validating the file schema.
// Then, it loads the schema and the document using the gojsonschema package.
// After that, it calls the Validate function to perform the schema validation.
// If there is an error while validating the schema, the function panics and logs an error message.
// If the file JSON is poorly formatted and does not pass the schema validation, the function constructs an error message with the filename and the validation errors.
// The error message is then returned as an error.
// If the JSON is valid and passes the schema validation, the function returns nil.
func validateJsonBySchema(fileJsonUri string, fileJsonBytes []byte) error {
	printInfoLogf("Validating the %s file schema...", fileJsonUri)

	// carregamos o schema e o documento
	schemaLoader := gojsonschema.NewReferenceLoader(gopenJsonSchema)
	documentLoader := gojsonschema.NewBytesLoader(fileJsonBytes)

	// chamamos o validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if helper.IsNotNil(err) {
		panic(errors.New("Error validate schema:", err))
	}

	// checamos se valido, caso nao seja formatamos a mensagem
	if !result.Valid() {
		errorMsg := fmt.Sprintf("Json %s poorly formatted!\n", fileJsonUri)
		for _, desc := range result.Errors() {
			errorMsg += fmt.Sprintf("- %s\n", desc)
		}
		return errors.New(errorMsg)
	}
	// se tudo ocorrem bem retornamos nil
	return nil
}

// buildCacheStore builds and configures a cache store based on the provided storeDTO.
// If storeDTO.Redis is not empty, it returns a new Redis cache store initialized with the Redis address and password.
// Otherwise, it returns a new Memory cache store.
func buildCacheStore(storeDTO *dto.Store) infra.CacheStore {
	printInfoLog("Configuring cache store...")
	if helper.IsNotNil(storeDTO) {
		return infra.NewRedisStore(storeDTO.Redis.Address, storeDTO.Redis.Password)
	}
	return infra.NewMemoryStore()
}

// listerAndServer initializes and runs the Gopen application with the provided cache store and Gopen configuration.
// It builds the necessary infrastructures, services, middlewares, controllers, and the Gopen application.
// The Gopen application is then started by calling its ListAndServer() method.
func listerAndServer(cacheStore infra.CacheStore, gopenVO *vo.Gopen) {
	printInfoLog("Building infra..")
	restTemplate := infra.NewRestTemplate()
	traceProvider := infra.NewTraceProvider()
	logProvider := infra.NewLogProvider()

	printInfoLog("Building domain..")
	modifierService := service.NewModifier()
	backendService := service.NewBackend(modifierService, restTemplate)
	endpointService := service.NewEndpoint(backendService)

	printInfoLog("Building middlewares..")
	traceMiddleware := middleware.NewTrace(traceProvider)
	logMiddleware := middleware.NewLog(logProvider)
	securityCorsMiddleware := middleware.NewSecurityCors(gopenVO.SecurityCors())
	limiterMiddleware := middleware.NewLimiter()
	timeoutMiddleware := middleware.NewTimeout()
	cacheMiddleware := middleware.NewCache(cacheStore)

	printInfoLog("Building controllers..")
	staticController := controller.NewStatic(gopenVO)
	endpointController := controller.NewEndpoint(endpointService)

	printInfoLog("Building application..")
	gopenApp = app.NewGopen(
		gopenVO,
		traceMiddleware,
		logMiddleware,
		securityCorsMiddleware,
		timeoutMiddleware,
		limiterMiddleware,
		cacheMiddleware,
		staticController,
		endpointController,
	)

	// chamamos o lister and server da aplicação
	gopenApp.ListerAndServer()
}

// writeGopenJsonResult writes the Gopen JSON result to a file.
//
// It marshals the GopenViewDTO built from the Gopen value object and Store
// DTO using the BuildGopenViewDTOFromCMD function from the mapper package
// into a JSON byte array. Then it writes the JSON byte array to a file
// with the filepath defined in the gopenJsonResult constant.
//
// If any error occurs during the marshaling or file write operation,
// it logs a warning with the error message using the printWarningLog function.
//
// Parameters:
// - gopenVO: The Gopen value object.
// - storeDTO: The Store DTO.
func writeGopenJsonResult(gopenVO *vo.Gopen, storeDTO *dto.Store) {
	gopenBytes, err := json.MarshalIndent(mapper.BuildGopenDTOFromCMD(gopenVO, storeDTO), "", "\t")
	if helper.IsNil(err) {
		err = os.WriteFile(gopenJsonResult, gopenBytes, 0644)
	}
	if helper.IsNotNil(err) {
		printWarningLogf("Error write file %s result: %s", gopenJsonResult, err)
	}
}

// configureWatcher configures and returns a watcher if hot reload is enabled in the GopenDTO.
// It creates a new fsnotify.Watcher, initializes a new goroutine to listen to events, and adds the .env and .json files to be watched.
// If any error occurs during the configuration, a warning log is printed.
//
// Parameters:
// - env: the environment string
// - gopenDTO: the GopenDTO containing the configuration
//
// Returns:
// - *fsnotify.Watcher: the configured watcher, or nil if hot reload is disabled
func configureWatcher(env string, gopenDTO *dto.Gopen) *fsnotify.Watcher {
	if !gopenDTO.HotReload {
		return nil
	}

	printInfoLog("Configuring watcher...")

	// instânciamos o novo watcher
	watcher, err := fsnotify.NewWatcher()
	if helper.IsNotNil(err) {
		printWarningLog("Error configure watcher:", err)
	}

	// inicializamos o novo goroutine de ouvinte de eventos
	go watchEvents(env, watcher)

	// adicionamos os arquivos a serem observados
	fileEnvUri := getFileEnvUri(env)
	fileJsonUri := getFileJsonUri(env)

	// primeiro tentamos adicionar o .env
	err = watcher.Add(fileEnvUri)
	if helper.IsNotNil(err) {
		printWarningLogf("Error add watcher on file: %s err: %s", fileEnvUri, err)
	}
	// depois tentamos adicionar o .json
	err = watcher.Add(fileJsonUri)
	if helper.IsNotNil(err) {
		printWarningLogf("Error add watcher on file: %s err: %s", fileJsonUri, err)
	}

	return watcher
}

// watchEvents opens an infinite loop to listen for events from the watcher.
// It waits for notifications from the watcher channel and executes the corresponding event or error handler.
func watchEvents(env string, watcher *fsnotify.Watcher) {
	// abrimos um for infinito para sempre ouvir os eventos do watcher
	for {
		// prendemos o loop atual aguardando o canal ser notificado de watcher
		select {
		case event, ok := <-watcher.Events:
			// chamamos a função que executa o evento
			executeEvent(env, event, ok)
			break
		case err, ok := <-watcher.Errors:
			// chamamos a função que executa o evento de erro
			executeErrorEvent(err, ok)
			break
		}
	}
}

// executeEvent executes an event triggered by file modification.
// Arguments:
// - env: the environment string.
// - event: the fsnotify.Event containing information about the event.
// - ok: a boolean indicating if the event is valid.
// If the event is not valid, the function returns immediately.
//
// The function prints a log message indicating the file modification event that was triggered.
//
// The function then calls the restartApp function passing the environment string.
func executeEvent(env string, event fsnotify.Event, ok bool) {
	if !ok {
		return
	}
	printInfoLogf("File modification event %s on file %s triggered!", event.Op.String(), event.Name)
	restartApp(env)
}

// executeErrorEvent handles the error event triggered by the watcher. If the ok parameter is false, it returns without
// executing any action. Otherwise, it logs a warning message with
func executeErrorEvent(err error, ok bool) {
	if !ok {
		return
	}
	printWarningLogf("Watcher event error triggered! err: %s", err)
}

// startApp configures the store interface, sets up the watcher
// for listening to configuration file changes, builds the value
// objects, and calls the listerAndServer function.
func startApp(env string, gopenDTO *dto.Gopen) {
	// configuramos o store interface
	cacheStore := buildCacheStore(gopenDTO.Store)
	defer closeCacheStore(cacheStore)

	// configuramos o watch para ouvir mudanças do json de configuração
	watcher := configureWatcher(env, gopenDTO)
	defer closeWatcher(watcher)

	// construímos os objetos de valores a partir do dto gopen
	printInfoLog("Building value objects..")
	gopenVO := vo.NewGopen(env, gopenDTO)

	// salvamos o gopenDTO resultante
	writeGopenJsonResult(gopenVO, gopenDTO.Store)

	// chamamos o lister and server, ele ira segurar a goroutine, depois que ele é parado, as linhas seguintes vão ser chamados
	listerAndServer(cacheStore, gopenVO)
}

// restartApp restarts the current application based on a specified
// environment. It performs several operations:
//  1. Shuts down the current server within a timeout of 30 seconds
//  2. If the shutdown is successful, it loads the environment variables
//     from the specified environment (denoted by the variable 'env')
//  3. It then loads the new Data Transfer Object (DTO) from the specified
//     environment
//  4. Lastly, starts a new app listener using the loaded information.
//
// The function receives a string 'env' as the required environment parameter.
// The environment setting loaded will determine the specific configurations
// to use when re-starting the application. It should be noted that the function
// does not return any value and will log relevant information and errors to
// the console as it executes.
func restartApp(env string) {
	// print log
	printInfoLog("---------- RESTART ----------")

	// inicializamos um contexto de timeout para ter um tempo de limite de tentativa
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// paramos a aplicação, para começar com o novo DTO e as novas envs
	printInfoLog("Shutting down current server...")
	err := gopenApp.Shutdown(ctx)
	if helper.IsNotNil(err) {
		printWarningLogf("Error shutdown app: %s!", err)
		return
	}

	// carregamos as variáveis de ambiente indicada
	loadGopenEnvs(env)

	// lemos o novo DTO
	gopenDTO := loadGopenJson(env)

	// começamos um novo app listener com as informações alteradas
	go startApp(env, gopenDTO)
}

// removeGopenJsonResult handles the removal of a JSON result file.
// This function will try to delete the file specified by the constant gopenJsonResult.
// If the file does not exist, the function will exit silently.
// If there is an error during removal that is NOT due to the file not existing,
// it logs a warning message with printWarningLogf.
func removeGopenJsonResult() {
	err := os.Remove(gopenJsonResult)
	if helper.IsNotNil(err) && errors.IsNot(err, os.ErrNotExist) {
		printWarningLogf("Error remove %s err: %s", gopenJsonResult, err)
		return
	}
}

// closeWatcher closes the fsnotify.Watcher if it is not nil.
// If there is an error while closing the watcher, it logs a warning message.
func closeWatcher(watcher *fsnotify.Watcher) {
	if helper.IsNotNil(watcher) {
		err := watcher.Close()
		if helper.IsNotNil(err) {
			printWarningLogf("Error close watcher: %s", err)
		}
	}
}

// closeCacheStore closes the cache store by calling the Close method on the provided infra.CacheStore object.
// If an error occurs during the close operation, a warning log is printed.
func closeCacheStore(store infra.CacheStore) {
	err := store.Close()
	if helper.IsNotNil(err) {
		printWarningLog("Error close cache store:", err)
	}
}

// getFileEnvUri returns the file URI for the given environment.
// The returned URI follows the format "./gopen/{env}.env".
func getFileEnvUri(env string) string {
	return fmt.Sprintf("./gopen/%s/.env", env)
}

// getFileJsonUri returns the file URI for the specified environment's JSON file.
// The returned URI follows the format "./gopen/{env}.json".
func getFileJsonUri(env string) string {
	return fmt.Sprintf("./gopen/%s/.json", env)
}

// printInfoLog prints an informational log message using the logger package.
func printInfoLog(msg ...any) {
	logger.InfoOpts(loggerOptions, msg...)
}

// printInfoLogf is a function that prints an information log message with formatting capabilities.
func printInfoLogf(format string, msg ...any) {
	logger.InfoOptsf(format, loggerOptions, msg...)
}

// printWarningLog prints a warning log message using the logger package.
func printWarningLog(msg ...any) {
	logger.WarningOpts(loggerOptions, msg...)
}

// printWarningLogf logs a warning message with the given format and arguments using the logger package.
func printWarningLogf(format string, msg ...any) {
	logger.WarningOptsf(format, loggerOptions, msg...)
}
