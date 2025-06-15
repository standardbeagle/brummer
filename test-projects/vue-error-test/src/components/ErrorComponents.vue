<template>
  <div class="error-components">
    <h2>Vue Error Test Components</h2>
    
    <!-- Template errors -->
    <div>
      <!-- Error: Property does not exist on this -->
      {{ nonExistentProperty }}
      
      <!-- Error: Cannot read properties of undefined -->
      {{ user.profile.name }}
      
      <!-- Error: Invalid directive -->
      <button v-invalid-directive="handleClick">Invalid Directive</button>
      
      <!-- Error: Method not found -->
      <button @click="nonExistentMethod">Missing Method</button>
    </div>
    
    <div>
      <!-- Error: v-for without key -->
      <div v-for="item in items">{{ item }}</div>
      
      <!-- Error: Invalid v-model -->
      <input v-model="invalidModel.deep.prop" />
    </div>
    
    <button @click="triggerRuntimeError">Trigger Runtime Error</button>
    <button @click="triggerAsyncError">Trigger Async Error</button>
    <button @click="triggerTypeError">Trigger Type Error</button>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'

// TypeScript errors
interface User {
  name: string;
  age: number;
}

// Error: Type 'string' is not assignable to type 'number'
const invalidUser: User = {
  name: "John",
  age: "thirty" as any
};

// Error: Property 'invalidProp' does not exist on type 'User'
const userWithError = reactive<User>({
  name: "Jane",
  age: 25,
  invalidProp: "error" as any
});

// Runtime error variables
const items = ref([1, 2, 3]);
const user = ref<User | null>(null);
const errorData = ref(null);

// Error: Computed property referencing undefined
const computedError = computed(() => {
  return (undefinedVariable as any).someProperty;
});

// Error: Watch with missing dependencies
watch(() => userWithError.name, (newVal) => {
  console.log(items.value.length); // items not in watch deps
});

// Methods with errors
const triggerRuntimeError = () => {
  try {
    // Error: Cannot read properties of null
    console.log((null as any).someProperty);
    
    // Error: TypeError accessing undefined
    const undefinedObj = undefined;
    console.log((undefinedObj as any).prop);
    
    // Error: ReferenceError
    console.log((window as any).undefinedGlobal);
    
  } catch (error) {
    console.error('Runtime error caught:', error);
  }
};

const triggerAsyncError = async () => {
  try {
    // Error: Network request to invalid URL
    const response = await fetch('https://invalid-api-endpoint.nonexistent');
    const data = await response.json();
    console.log(data);
  } catch (error) {
    console.error('Async error:', error);
    throw new Error('Custom Vue async error');
  }
};

const triggerTypeError = () => {
  // Error: Type assertions causing runtime errors
  const stringVar = "hello";
  const numberResult = (stringVar as any) / 2;
  console.log(numberResult); // NaN
  
  // Error: Calling non-function
  const notAFunction = "string";
  try {
    (notAFunction as any)();
  } catch (error) {
    console.error('Type error:', error);
  }
};

// Lifecycle hook with errors
onMounted(() => {
  // Error: Accessing properties before data is loaded
  console.log(user.value.name); // user is null
  
  // Error: Promise rejection
  Promise.reject(new Error('Vue component mount error'));
  
  // Error: Unhandled promise
  new Promise((resolve, reject) => {
    reject(new Error('Unhandled Vue promise rejection'));
  });
});

// Error: Invalid reactive assignment
// const invalidReactive = reactive("string"); // Should be object

// Error: Ref type mismatch
const typedRef = ref<number>(0);
// typedRef.value = "string"; // Would cause TypeScript error
</script>

<style scoped>
.error-components {
  padding: 20px;
  border: 1px solid #ccc;
  margin: 10px;
}

/* CSS error */
.invalid-css {
  invalid-property: invalid-value;
  color: #invalid-color;
}
</style>