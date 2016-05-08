class Person < ActiveRecord::Migration
  def change
    create_table :person do |t|
      t.string :name
      t.integer :age
    end
  end
end
