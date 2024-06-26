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

package vo

// modifyParam represents the parameters for modifying an entity. It contains a modify field
// which defines the modifications to be applied.
type modifyParam struct {
	modify
}

// NewModifyParam creates a new instance of modifyParam struct
// with the provided Modifier, Request, and Response.
func NewModifyParam(modifierVO *Modifier, requestVO *Request, responseVO *Response) ModifierStrategy {
	return modifyParam{
		modify: newModify(modifierVO, requestVO, responseVO),
	}
}

// Execute executes the modifyParam by calling the executeRequestScope method and returns the modified Request and Response.
// The execution starts from the default scope.
func (m modifyParam) Execute() (*Request, *Response) {
	// executamos a partir do escopo padrão
	return m.executeRequestScope()
}

// executeRequestScope executes request scope modifications.
// It first modifies local and global parameters, then modifies local request with updated local parameters,
// following by modification of global request with updated global parameters. It also returns a response.
// The function returns modified Request and Response after the modifications are done.
func (m modifyParam) executeRequestScope() (*Request, *Response) {
	// chamamos o modify de params passando o path e params a ser modificado e o mesmo retorna os mesmo modificados
	localPath, globalParams, localParams := m.params(m.localRequestPath(), m.globalRequestParams(), m.localRequestParams())

	// modificamos o params local
	backendRequestVO := m.modifyLocalRequest(localPath, localParams)

	// modificamos o params propagate e retornamos
	return m.modifyGlobalRequest(globalParams, backendRequestVO), m.response
}

// localRequestPath returns the path of the current backend request in the modifyParam struct.
func (m modifyParam) localRequestPath() UrlPath {
	return m.request.CurrentBackendRequest().Path()
}

// globalRequestParams returns the propagate request parameters of the request object in the modifyParam struct.
func (m modifyParam) globalRequestParams() Params {
	return m.request.Params()
}

// localRequestParams returns the local request parameters of the current backend request object in the modifyParam struct.
func (m modifyParam) localRequestParams() Params {
	return m.request.CurrentBackendRequest().Params()
}

// modifyLocalRequest is a method on the modifyParam type.
// This method takes in a 'localPath' of type string and 'localParams' of type Params,
// and returns a modified backendRequest.
//
// The 'localPath' represents the local path of the backend request.
//
// The 'localParams' represents the new local parameters to be used for the backend request.
//
// This method is used to modify the parameters of a local backend request with new parameters.
func (m modifyParam) modifyLocalRequest(localPath UrlPath, localParams Params) *backendRequest {
	return m.request.CurrentBackendRequest().ModifyParams(localPath, localParams)
}

// modifyGlobalRequest is a method on the modifyParam struct.
// It modifies propagate request parameters based on provided Params and backendRequest.
// It takes two arguments - globalParams of type Params and backendRequestVO of type backendRequest.
// It returns a Request which is the modified version of the original Request.
//
// Parameters:
//
// globalParams: The Params type propagate parameters that need to be modified.
//
// backendRequestVO: The backendRequest type which contains the request sent to the backend.
//
// Returns:
//
// The modified Request after applying changes based on globalParams and backendRequestVO.
func (m modifyParam) modifyGlobalRequest(globalParams Params, backendRequestVO *backendRequest) *Request {
	return m.request.ModifyParams(globalParams, backendRequestVO)
}
