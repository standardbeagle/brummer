import React, { useState, useEffect } from 'react';

// Component with TypeScript errors
export const TypeScriptErrorComponent: React.FC = () => {
  // Error: Type 'string' is not assignable to type 'number'
  const [count, setCount] = useState<number>("invalid");
  
  // Error: Property 'nonExistent' does not exist on type '{}'
  const badObject = {};
  console.log(badObject.nonExistent);
  
  // Error: JSX element implicitly has type 'any'
  const BadJSXElement = () => <div>{unknownVariable}</div>;
  
  return <div>TypeScript Error Component</div>;
};

// Component with runtime errors
export const RuntimeErrorComponent: React.FC = () => {
  const [data, setData] = useState(null);
  
  useEffect(() => {
    // Error: Cannot read properties of null
    setData(null);
    console.log(data.someProperty);
    
    // Error: TypeError: Cannot read properties of undefined
    const undefinedObj = undefined;
    console.log(undefinedObj.prop);
    
    // Error: ReferenceError: undefinedVariable is not defined
    console.log(undefinedVariable);
    
    // Error: fetch request to invalid URL
    fetch('invalid-url').catch(error => {
      console.error('Fetch error:', error);
    });
  }, []);
  
  return <div>Runtime Error Component</div>;
};

// Component with async errors
export const AsyncErrorComponent: React.FC = () => {
  const [loading, setLoading] = useState(false);
  
  const handleAsyncError = async () => {
    setLoading(true);
    try {
      // Error: Network request failure
      const response = await fetch('https://invalid-api-endpoint.nonexistent');
      const data = await response.json();
      console.log(data);
    } catch (error) {
      console.error('Async operation failed:', error);
      throw new Error('Custom async error message');
    } finally {
      setLoading(false);
    }
  };
  
  // Error: Unhandled promise rejection
  const unhandledPromise = () => {
    Promise.reject(new Error('Unhandled promise rejection'));
  };
  
  return (
    <div>
      <button onClick={handleAsyncError}>Trigger Async Error</button>
      <button onClick={unhandledPromise}>Trigger Unhandled Promise</button>
    </div>
  );
};

// Component with JSX errors
export const JSXErrorComponent: React.FC = () => {
  const items = [1, 2, 3];
  
  return (
    <div>
      {/* Error: Missing key prop */}
      {items.map(item => <div>{item}</div>)}
      
      {/* Error: Adjacent JSX elements must be wrapped */}
      <span>First</span>
      <span>Second</span>
      
      {/* Error: Objects are not valid as React children */}
      {items}
      
      {/* Error: Functions are not valid as React children */}
      {() => "function as child"}
    </div>
  );
};

// Component with hook errors
export const HookErrorComponent: React.FC = () => {
  const [count, setCount] = useState(0);
  
  // Error: React Hook useEffect is called conditionally
  if (count > 5) {
    useEffect(() => {
      console.log('Conditional effect');
    }, []);
  }
  
  // Error: React Hook has a missing dependency
  useEffect(() => {
    console.log(count);
  }, []); // count is missing from dependency array
  
  return <div>Hook Error Component</div>;
};

// Component with build/compilation errors
export const BuildErrorComponent: React.FC = () => {
  // Error: Cannot find module
  import invalidModule from 'non-existent-module';
  
  // Error: Syntax error
  const invalidSyntax = {
    prop1: "value1"
    prop2: "value2" // Missing comma
  };
  
  // Error: Type error
  const numberVar: number = "string value";
  
  return <div>Build Error Component</div>;
};