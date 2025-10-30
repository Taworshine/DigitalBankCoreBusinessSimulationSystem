import { useState, useEffect } from 'react';

// 子组件示例
function UserGreeting({ userName, onLogout }) {
  return (
    <div className="greeting">
      <h3>欢迎回来，{userName}！</h3>
      <button onClick={onLogout}>退出登录</button>
    </div>
  );
}

// 列表项组件
function TodoItem({ todo, onToggle, onDelete }) {
  return (
    <li style={{ textDecoration: todo.completed ? 'line-through' : 'none' }}>
      <input
        type="checkbox"
        checked={todo.completed}
        onChange={() => onToggle(todo.id)}
      />
      {todo.text}
      <button onClick={() => onDelete(todo.id)}>删除</button>
    </li>
  );
}

// 主应用组件
function App() {
  // 状态管理 - 使用useState钩子
  const [userName, setUserName] = useState('');
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [todos, setTodos] = useState([]);
  const [newTodo, setNewTodo] = useState('');
  const [currentTime, setCurrentTime] = useState(new Date());

  // 副作用 - 使用useEffect钩子（模拟数据加载和定时器）
  useEffect(() => {
    // 组件挂载时加载初始数据
    const savedTodos = localStorage.getItem('todos');
    if (savedTodos) {
      setTodos(JSON.parse(savedTodos));
    }

    // 定时器更新时间
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    // 清理函数 - 组件卸载时执行
    return () => clearInterval(timer);
  }, []); // 空依赖数组表示只在挂载和卸载时执行

  // 监听todos变化，保存到localStorage
  useEffect(() => {
    localStorage.setItem('todos', JSON.stringify(todos));
  }, [todos]); // 依赖todos，当todos变化时执行

  // 事件处理函数 - 登录
  const handleLogin = (e) => {
    e.preventDefault(); // 阻止表单默认提交行为
    if (userName.trim()) {
      setIsLoggedIn(true);
    }
  };

  // 事件处理函数 - 登出
  const handleLogout = () => {
    setIsLoggedIn(false);
    setUserName('');
  };

  // 事件处理函数 - 添加待办事项
  const handleAddTodo = (e) => {
    e.preventDefault();
    if (newTodo.trim()) {
      const newItem = {
        id: Date.now(), // 使用时间戳作为唯一ID
        text: newTodo,
        completed: false
      };
      // 更新数组状态（不可变方式）
      setTodos([...todos, newItem]);
      setNewTodo(''); // 清空输入框
    }
  };

  // 切换待办事项状态
  const toggleTodo = (id) => {
    setTodos(
      todos.map(todo => 
        todo.id === id ? { ...todo, completed: !todo.completed } : todo
      )
    );
  };

  // 删除待办事项
  const deleteTodo = (id) => {
    setTodos(todos.filter(todo => todo.id !== id));
  };

  // 计算已完成的待办事项数量
  const completedCount = todos.filter(todo => todo.completed).length;

  return (
    <div className="app" style={{ maxWidth: '600px', margin: '0 auto', padding: '20px' }}>
      <h1>React 示例应用</h1>
      <p>当前时间: {currentTime.toLocaleTimeString()}</p>

      {/* 条件渲染 - 根据登录状态显示不同内容 */}
      {isLoggedIn ? (
        <UserGreeting userName={userName} onLogout={handleLogout} />
      ) : (
        <form onSubmit={handleLogin} style={{ marginBottom: '20px' }}>
          <input
            type="text"
            placeholder="请输入用户名"
            value={userName}
            onChange={(e) => setUserName(e.target.value)} // 受控组件
            style={{ marginRight: '10px', padding: '5px' }}
          />
          <button type="submit">登录</button>
        </form>
      )}

      {isLoggedIn && ( // 只有登录后才显示待办事项区域
        <div className="todo-section">
          <h2>待办事项列表</h2>
          <form onSubmit={handleAddTodo} style={{ marginBottom: '10px' }}>
            <input
              type="text"
              placeholder="添加新的待办事项..."
              value={newTodo}
              onChange={(e) => setNewTodo(e.target.value)} // 受控组件
              style={{ marginRight: '10px', padding: '5px', width: '300px' }}
            />
            <button type="submit">添加</button>
          </form>

          {/* 列表渲染 - 使用map生成列表 */}
          <ul style={{ listStyle: 'none', paddingLeft: 0 }}>
            {todos.length === 0 ? (
              <li>暂无待办事项</li>
            ) : (
              todos.map(todo => (
                <TodoItem
                  key={todo.id} // 列表项必须有唯一key
                  todo={todo}
                  onToggle={toggleTodo}
                  onDelete={deleteTodo}
                />
              ))
            )}
          </ul>

          {/* 显示统计信息 */}
          <p>已完成: {completedCount} / 总计: {todos.length}</p>
        </div>
      )}
    </div>
  );
}

export default App;