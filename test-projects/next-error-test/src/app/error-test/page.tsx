'use client';

import React, { useState, useEffect } from 'react';

// Next.js Error Test Page
export default function ErrorTestPage() {
  const [data, setData] = useState(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Trigger various errors on page load
    triggerInitialErrors();
  }, []);

  const triggerInitialErrors = () => {
    // Error: Cannot read properties of null
    try {
      console.log((null as any).someProperty);
    } catch (err) {
      console.error('Null property access error:', err);
    }

    // Error: Undefined variable access
    try {
      console.log((window as any).undefinedGlobal);
    } catch (err) {
      console.error('Undefined variable error:', err);
    }

    // Error: Type coercion issues
    const result = "string" / 2;
    console.log('Type coercion result (NaN):', result);
  };

  const triggerNetworkError = async () => {
    try {
      // Error: Network request to invalid endpoint
      const response = await fetch('/api/nonexistent-endpoint');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setData(data);
    } catch (err) {
      console.error('Network error:', err);
      setError('Failed to fetch data from API');
    }
  };

  const triggerClientSideError = () => {
    // Error: Accessing DOM elements that don't exist
    try {
      const element = document.getElementById('nonexistent-element');
      console.log(element!.innerHTML); // Force non-null assertion error
    } catch (err) {
      console.error('DOM access error:', err);
    }

    // Error: localStorage access in SSR context (would fail during build)
    try {
      localStorage.setItem('test', 'value');
    } catch (err) {
      console.error('localStorage error:', err);
    }
  };

  const triggerUnhandledPromise = () => {
    // Error: Unhandled promise rejection
    Promise.reject(new Error('Next.js unhandled promise rejection'));
    
    // Error: Promise chain without catch
    fetch('https://invalid-domain.nonexistent')
      .then(response => response.json())
      .then(data => console.log(data));
      // Missing .catch() handler
  };

  const triggerTypeScriptError = () => {
    // TypeScript compilation errors (would show during build)
    
    // Error: Type 'string' is not assignable to type 'number'
    const numberVar: number = "string" as any;
    
    // Error: Property 'nonExistent' does not exist
    const obj = { name: "test" };
    console.log((obj as any).nonExistent);
    
    // Error: Cannot invoke an object which is possibly undefined
    const maybeFunction: (() => void) | undefined = undefined;
    try {
      maybeFunction!(); // Force invocation
    } catch (err) {
      console.error('Function invocation error:', err);
    }
  };

  const triggerRenderError = () => {
    // Error: Setting state that will cause render errors
    setData("invalid data format" as any);
  };

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-4">Next.js Error Test Page</h1>
      
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          Error: {error}
        </div>
      )}

      <div className="space-y-4">
        <button 
          onClick={triggerNetworkError}
          className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
        >
          Trigger Network Error
        </button>

        <button 
          onClick={triggerClientSideError}
          className="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600"
        >
          Trigger Client-Side Error
        </button>

        <button 
          onClick={triggerUnhandledPromise}
          className="bg-yellow-500 text-white px-4 py-2 rounded hover:bg-yellow-600"
        >
          Trigger Unhandled Promise
        </button>

        <button 
          onClick={triggerTypeScriptError}
          className="bg-purple-500 text-white px-4 py-2 rounded hover:bg-purple-600"
        >
          Trigger TypeScript Errors
        </button>

        <button 
          onClick={triggerRenderError}
          className="bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600"
        >
          Trigger Render Error
        </button>
      </div>

      {/* Component with intentional errors */}
      <ErrorComponent data={data} />
    </div>
  );
}

// Component with various error patterns
function ErrorComponent({ data }: { data: any }) {
  const [items] = useState([1, 2, 3]);

  return (
    <div className="mt-8 p-4 border rounded">
      <h2 className="text-xl font-semibold mb-2">Error Component</h2>
      
      {/* Error: Missing key props */}
      {items.map(item => (
        <div>{item}</div>
      ))}
      
      {/* Error: Conditional rendering without null check */}
      <p>Data length: {data.length}</p>
      
      {/* Error: Accessing nested properties without validation */}
      <p>Nested data: {data.user.profile.name}</p>
    </div>
  );
}