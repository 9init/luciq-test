class Application < ApplicationRecord
  has_many :chats, dependent: :destroy
  validates :name, presence: true
  validates :token, presence: true, uniqueness: true
  before_validation :generate_token, on: :create

  private

  def generate_token
    self.token ||= SecureRandom.uuid
  end
end
