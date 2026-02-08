// enemy.go: Pure Go core type.
//
// This file defines the Enemy entity. It owns all mutable state,
// enforces invariants, and knows nothing about Lua.
package main

// Enemy is a game entity with health, position, and alive state.
// Go owns this state; Lua may read and mutate it only through
// explicit method calls.
type Enemy struct {
	name   string
	health int
	maxHP  int
	x, y   float64
	alive  bool
}

// NewEnemy creates an Enemy with full health at the given position.
func NewEnemy(name string, maxHP int, x, y float64) *Enemy {
	return &Enemy{
		name:   name,
		health: maxHP,
		maxHP:  maxHP,
		x:      x,
		y:      y,
		alive:  true,
	}
}

// Name returns the enemy's name.
func (e *Enemy) Name() string { return e.name }

// Health returns current health.
func (e *Enemy) Health() int { return e.health }

// MaxHP returns maximum health.
func (e *Enemy) MaxHP() int { return e.maxHP }

// Position returns the enemy's (x, y) coordinates.
func (e *Enemy) Position() (float64, float64) { return e.x, e.y }

// IsAlive returns whether the enemy is still alive.
func (e *Enemy) IsAlive() bool { return e.alive }

// TakeDamage reduces health by amount (clamped to 0).
// If health reaches 0, the enemy dies.
// Does nothing if already dead.
func (e *Enemy) TakeDamage(amount int) {
	if !e.alive || amount <= 0 {
		return
	}
	e.health -= amount
	if e.health <= 0 {
		e.health = 0
		e.alive = false
	}
}

// Heal restores health by amount (clamped to maxHP).
// Does nothing if dead.
func (e *Enemy) Heal(amount int) {
	if !e.alive || amount <= 0 {
		return
	}
	e.health += amount
	if e.health > e.maxHP {
		e.health = e.maxHP
	}
}

// MoveTo sets the enemy's position.
// Does nothing if dead.
func (e *Enemy) MoveTo(x, y float64) {
	if !e.alive {
		return
	}
	e.x = x
	e.y = y
}
