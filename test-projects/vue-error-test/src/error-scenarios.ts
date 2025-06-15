// TypeScript file with Vue-specific error scenarios

console.log('Vue error scenarios starting...');

// 1. Vue compilation errors
interface User {
  name: string;
  age: number;
}

// TypeScript error: Type 'string' is not assignable to type 'number'
const invalidUser: User = {
  name: "John",
  age: "thirty" as any // This will cause runtime issues
};

// 2. Reactive system errors
import { ref, reactive, computed } from 'vue';

const userData = ref<User | null>(null);

// Error: Trying to access properties of null
try {
  console.log(userData.value!.name);
} catch (error) {
  console.error('Vue Ref Error:', error);
}

// 3. Computed property errors
const computedValue = computed(() => {
  // Error: Accessing undefined property
  return (userData.value as any).profile.details.email;
});

try {
  console.log(computedValue.value);
} catch (error) {
  console.error('Vue Computed Error:', error);
}

// 4. Watchers with errors
import { watch } from 'vue';

watch(() => userData.value, (newValue) => {
  // Error: Accessing properties without null check
  console.log(newValue!.name.toUpperCase());
}, { immediate: true });

// 5. Async component errors
async function loadUserData() {
  try {
    const response = await fetch('https://invalid-vue-api.nonexistent/users');
    const data = await response.json();
    userData.value = data;
  } catch (error) {
    console.error('Vue Async Error:', error);
    throw new Error('Failed to load user data in Vue component');
  }
}

loadUserData().catch(error => {
  console.error('Unhandled Vue async error:', error);
});

// 6. Event handler errors
export function handleUserClick() {
  try {
    // Error: Method call on undefined
    (userData.value as any).save();
  } catch (error) {
    console.error('Vue Event Handler Error:', error);
  }
}

// 7. Template compilation errors (would show during build)
// These would cause compilation failures:
// - Missing closing tags
// - Invalid directive names
// - Incorrect v-for syntax

// 8. Router errors
try {
  // Simulating router navigation error
  throw new Error('NavigationDuplicated: Avoided redundant navigation to current location');
} catch (error) {
  console.error('Vue Router Error:', error);
}

// 9. Store/Pinia errors
const invalidStoreAction = () => {
  try {
    // Error: Accessing store before initialization
    const store = null;
    (store as any).dispatch('nonexistentAction');
  } catch (error) {
    console.error('Vue Store Error:', error);
  }
};

invalidStoreAction();

// 10. Lifecycle hook errors
export function simulateLifecycleError() {
  try {
    // Error: Calling lifecycle hook outside component
    (null as any).onMounted(() => {
      console.log('This will fail');
    });
  } catch (error) {
    console.error('Vue Lifecycle Error:', error);
  }
}

simulateLifecycleError();

console.log('Vue error scenarios completed.');