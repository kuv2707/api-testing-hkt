<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { reactive, ref } from 'vue'
import { Edit, Delete } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { Pair } from './types'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const emptyStore = function() {
  return {
    name: '',
    url: '',
    username: '',
    password: '',
    kind: {
        name: '',
        url: ''
    },
    properties: [{
        key: '',
        value: ''
    }]
  } as Store
}
const stores = ref([] as Store[])
const dialogVisible = ref(false)
const creatingLoading = ref(false)
const storeFormRef = ref<FormInstance>()
const createAction = ref(true)
const storeForm = reactive(emptyStore())

interface Store {
  name: string
  url: string
  username: string
  password: string
  ready: boolean
  kind: {
    name: string
    url: string
  }
  properties: Pair[]
}

function loadStores() {
  const requestOptions = {
    method: 'POST',
  }
  fetch('/server.Runner/GetStores', requestOptions)
    .then((response) => response.json())
    .then((e) => {
      stores.value = e.data
    })
    .catch((e) => {
      ElMessage.error('Oops, ' + e)
    })
}
loadStores()

function deleteStore(name: string) {
  const requestOptions = {
    method: 'POST',
    body: JSON.stringify({
      name: name
    })
  }
  fetch('/server.Runner/DeleteStore', requestOptions)
    .then((response) => response.json())
    .then((e) => {
      ElMessage({
        message: 'Deleted.',
        type: 'success'
      })
      loadStores()
    })
    .catch((e) => {
      ElMessage.error('Oops, ' + e)
    })
}

function editStore(name: string) {
    dialogVisible.value = true
    stores.value.forEach((e: Store) => {
        if (e.name === name) {
            setStoreForm(e)
        }
    })
    createAction.value = false
}

function setStoreForm(store: Store) {
    storeForm.name = store.name
    storeForm.url = store.url
    storeForm.username = store.username
    storeForm.password = store.password
    storeForm.kind = store.kind
    storeForm.properties = store.properties
}

function addStore() {
    setStoreForm(emptyStore())
    dialogVisible.value = true
    createAction.value = true
}

const rules = reactive<FormRules<Store>>({
  name: [{ required: true, message: 'Name is required', trigger: 'blur' }]
})
const submitForm = async (formEl: FormInstance | undefined) => {
  if (!formEl) return
  await formEl.validate((valid: boolean, fields) => {
    if (valid) {
      creatingLoading.value = true

      const requestOptions = {
        method: 'POST',
        body: JSON.stringify(storeForm)
      }
      
      let api = '/server.Runner/CreateStore'
      if (!createAction.value) {
        api = '/server.Runner/UpdateStore'
      }

      fetch(api, requestOptions)
        .then((response) => {
          if (!response.ok) {
            throw new Error(response.statusText)
          } else {
            return response.json()
          }
        })
        .then(() => {
          loadStores()
          dialogVisible.value = false
          formEl.resetFields()
        })
        .catch((e) => {
          ElMessage.error('Oops, ' + e)
        })
        creatingLoading.value = false
    }
  })
}

function storeVerify(formEl: FormInstance | undefined) {
  if (!formEl) return

  const requestOptions = {
    method: 'POST',
    body: JSON.stringify({
      name: storeForm.name
    })
  }
  
  let api = '/server.Runner/VerifyStore'
  fetch(api, requestOptions)
    .then((response) => {
      if (!response.ok) {
        throw new Error(response.statusText)
      } else {
        return response.json()
      }
    })
    .then((e) => {
      if (e.success) {
        ElMessage({
          message: 'Verified!',
          type: 'success'
        })
      } else {
        ElMessage.error(e.message)
      }
    })
    .catch((e) => {
      ElMessage.error('Oops, ' + e)
    })
}

function updateKeys() {
  const props = storeForm.properties
  let lastItem = props[props.length - 1]
  if (lastItem.key !== '') {
    storeForm.properties.push({
      key: '',
      value: ''
    })
  }
}
</script>

<template>
    <div>Store Manager</div>
    <div>
        <el-button type="primary" @click="addStore" :icon="Edit">{{t('button.new')}}</el-button>
        <el-button type="primary" @click="loadStores">{{t('button.refresh')}}</el-button>
    </div>
    <el-table :data="stores" style="width: 100%">
      <el-table-column :label="t('field.name')" width="180">
        <template #default="scope">
          <el-input v-model="scope.row.name" placeholder="Name"/>
        </template>
      </el-table-column>
      <el-table-column label="URL">
        <template #default="scope">
          <div style="display: flex; align-items: center">
            <el-input v-model="scope.row.url" placeholder="URL" />
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('field.plugin')">
        <template #default="scope">
          <div style="display: flex; align-items: center">
            <el-input v-model="scope.row.kind.url" placeholder="Plugin" />
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('field.status')" width="100">
        <template #default="scope">
          <div style="display: flex; align-items: center">
            <el-text class="mx-1"
            type="success"
            v-if="scope.row.ready"
            >Ready</el-text>
            <el-text class="mx-1"
            type="warning"
            v-if="!scope.row.ready"
            >Not Ready</el-text>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('field.operations')" width="220">
        <template #default="scope">
          <div style="display: flex; align-items: center" v-if="scope.row.name !== 'local'">
            <el-button type="primary" @click="deleteStore(scope.row.name)" :icon="Delete">{{t('button.delete')}}</el-button>
            <el-button type="primary" @click="editStore(scope.row.name)" :icon="Edit">{{t('button.edit')}}</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div style="margin-top: 20px; margin-bottom: 20px; position: absolute; bottom: 0px;">
      Follow <el-link href="https://linuxsuren.github.io/api-testing/#storage" target="_blank">the instructions</el-link> to configure the storage plugins.
    </div>

    <el-dialog v-model="dialogVisible" :title="t('title.createStore')" width="30%" draggable>
      <template #footer>
      <span class="dialog-footer">
        <el-form
          :rules="rules"
          :model="storeForm"
          ref="storeFormRef"
          status-icon label-width="120px">
          <el-form-item :label="t('field.name')" prop="name">
            <el-input v-model="storeForm.name" test-id="store-form-name" />
          </el-form-item>
          <el-form-item label="URL" prop="url">
            <el-input v-model="storeForm.url" placeholder="http://foo" test-id="store-form-url" />
          </el-form-item>
          <el-form-item :label="t('field.username')" prop="username">
            <el-input v-model="storeForm.username" test-id="store-form-username" />
          </el-form-item>
          <el-form-item :label="t('field.password')" prop="password">
            <el-input v-model="storeForm.password" type="password" test-id="store-form-password" />
          </el-form-item>
          <el-form-item :label="t('field.pluginName')" prop="pluginName">
            <el-input v-model="storeForm.kind.name" test-id="store-form-plugin-name" />
          </el-form-item>
          <el-form-item :label="t('field.pluginURL')" prop="plugin">
            <el-input v-model="storeForm.kind.url" test-id="store-form-plugin" />
          </el-form-item>
          <el-form-item :label="t('field.properties')" prop="properties">
            <el-table :data="storeForm.properties" style="width: 100%">
                <el-table-column label="Key" width="180">
                    <template #default="scope">
                        <el-input v-model="scope.row.key" placeholder="Key" @change="updateKeys"/>
                    </template>
                </el-table-column>
                <el-table-column label="Value">
                    <template #default="scope">
                    <div style="display: flex; align-items: center">
                        <el-input v-model="scope.row.value" placeholder="Value" />
                    </div>
                    </template>
                </el-table-column>
            </el-table>
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              @click="storeVerify(storeFormRef)"
              test-id="store-form-verify"
              >{{t('button.verify')}}</el-button
            >
            <el-button
              type="primary"
              @click="submitForm(storeFormRef)"
              :loading="creatingLoading"
              test-id="store-form-submit"
              >{{t('button.submit')}}</el-button
            >
          </el-form-item>
        </el-form>
      </span>
    </template>
  </el-dialog>
</template>
